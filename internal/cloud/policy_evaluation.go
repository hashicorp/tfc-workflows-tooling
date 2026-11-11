// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-tfe"
	"github.com/sethvargo/go-retry"
)

// RunPolicyChecks is the include option for policy checks relationship.
// Note: This is not defined in go-tfe SDK as of v1.95.0
const RunPolicyChecks tfe.RunIncludeOpt = "policy_checks"

const (
	PolicyWaitMaxDuration    = 30 * time.Minute
	PolicyWaitInitialBackoff = 10 * time.Second
	PolicyWaitMaxBackoff     = 30 * time.Second
)

// PolicyService handles Sentinel policy operations for TFC/TFE runs
type PolicyService interface {
	// GetPolicyEvaluation retrieves policy evaluation results for a run.
	// Returns normalized PolicyEvaluation regardless of API format (legacy or modern).
	// Automatically waits (with retry) for policy evaluation to complete unless NoWait is true.
	GetPolicyEvaluation(ctx context.Context, options GetPolicyEvaluationOptions) (*PolicyEvaluation, error)

	// OverridePolicy applies a policy override with justification.
	// Pre-conditions: Run status must be post_plan_awaiting_decision.
	OverridePolicy(ctx context.Context, options OverridePolicyOptions) (*PolicyOverride, error)
}

// policyService implements PolicyService using go-tfe SDK
type policyService struct {
	*cloudMeta
}

// NewPolicyService creates a new policy service instance
func NewPolicyService(meta *cloudMeta) PolicyService {
	return &policyService{cloudMeta: meta}
}

// GetPolicyEvaluation retrieves policy evaluation results for a run
func (s *policyService) GetPolicyEvaluation(ctx context.Context, options GetPolicyEvaluationOptions) (*PolicyEvaluation, error) {
	// Validate options
	if err := options.Validate(); err != nil {
		return nil, err
	}

	// Read run to get workspace relationship and check if policies are ready
	run, err := s.tfe.Runs.ReadWithOptions(ctx, options.RunID, &tfe.RunReadOptions{
		Include: []tfe.RunIncludeOpt{tfe.RunWorkspace, tfe.RunTaskStages},
	})
	if err != nil {
		log.Printf("[ERROR] Failed to read run %s: %s", options.RunID, err)
		return nil, fmt.Errorf("error reading run: %w", err)
	}

	// Wait for policy evaluation to complete (unless NoWait is true)
	if !options.NoWait {
		run, err = s.waitForPolicyEvaluation(ctx, run)
		if err != nil {
			return nil, err
		}
	}

	// Try modern API first (task-stages/policy-evaluations)
	taskStages, err := s.tfe.TaskStages.List(ctx, run.ID, &tfe.TaskStageListOptions{})
	if err == nil && taskStages != nil && len(taskStages.Items) > 0 {
		log.Printf("[DEBUG] Using modern API (task-stages) for policy evaluation")
		result, err := s.getPolicyFromTaskStages(ctx, run, taskStages)
		if err == nil {
			return result, nil
		}
		// If no policy stage found in task stages, fall back to legacy API
		if !errors.Is(err, ErrNoPolicyCheck) {
			return nil, err
		}
		log.Printf("[DEBUG] No policy stage in task stages, trying legacy API")
	}

	// Fall back to legacy API (policy-checks)
	log.Printf("[DEBUG] Task stages not available, falling back to legacy policy-checks API")

	// Read the run again to get policy checks relationship
	run, err = s.tfe.Runs.ReadWithOptions(ctx, options.RunID, &tfe.RunReadOptions{
		Include: []tfe.RunIncludeOpt{RunPolicyChecks},
	})
	if err != nil {
		return nil, fmt.Errorf("error reading run for policy checks: %w", err)
	}

	if len(run.PolicyChecks) == 0 {
		log.Printf("[ERROR] No policy check or task stage found for run %s", run.ID)
		return nil, ErrNoPolicyCheck
	}

	policyCheck := run.PolicyChecks[0]

	return s.getPolicyFromPolicyCheck(ctx, run, policyCheck), nil
}

// waitForPolicyEvaluation polls until policy evaluation completes
func (s *policyService) waitForPolicyEvaluation(ctx context.Context, run *tfe.Run) (*tfe.Run, error) {
	// Check if policy evaluation is already complete
	if s.isPolicyEvaluationComplete(run) {
		return run, nil
	}

	log.Printf("[INFO] Waiting for policy evaluation to complete for run %s", run.ID)

	backoff := policyWaitBackoffStrategy()
	var finalRun *tfe.Run

	err := retry.Do(ctx, backoff, func(ctx context.Context) error {
		var err error
		finalRun, err = s.tfe.Runs.Read(ctx, run.ID)
		if err != nil {
			return fmt.Errorf("error reading run: %w", err)
		}

		log.Printf("[DEBUG] Polling run status: %s", finalRun.Status)

		// Check if policies are ready
		if s.isPolicyEvaluationComplete(finalRun) {
			return nil
		}

		// Check for terminal states that indicate policies won't be evaluated
		switch finalRun.Status {
		case tfe.RunDiscarded, tfe.RunCanceled, tfe.RunErrored:
			return fmt.Errorf("run entered terminal state %s without policy evaluation", finalRun.Status)
		}

		// Still waiting
		return retry.RetryableError(fmt.Errorf("policy evaluation still pending"))
	})

	if err != nil {
		return nil, err
	}

	return finalRun, nil
}

// isPolicyEvaluationComplete checks if policy evaluation is ready
func (s *policyService) isPolicyEvaluationComplete(run *tfe.Run) bool {
	// Check for statuses that indicate policies have been evaluated
	switch run.Status {
	case PostPlanAwaitingDecision, // Policies evaluated, awaiting decision
		tfe.RunPolicyOverride,     // Override applied
		tfe.RunPostPlanCompleted,  // Policies passed
		tfe.RunPlannedAndFinished, // Plan-only mode, policies evaluated
		tfe.RunPolicyChecked,      // Policies checked (confirmable run)
		tfe.RunPolicySoftFailed:   // Policies with advisory failures
		return true
	}
	return false
}

// policyWaitBackoffStrategy returns retry backoff configuration
func policyWaitBackoffStrategy() retry.Backoff {
	backoff := retry.NewFibonacci(PolicyWaitInitialBackoff)
	backoff = retry.WithCappedDuration(PolicyWaitMaxBackoff, backoff)
	backoff = retry.WithMaxDuration(PolicyWaitMaxDuration, backoff)
	return backoff
}

// getPolicyFromTaskStages extracts policy evaluation from modern API
func (s *policyService) getPolicyFromTaskStages(ctx context.Context, run *tfe.Run, taskStages *tfe.TaskStageList) (*PolicyEvaluation, error) {
	// Find policy evaluation stage
	var policyStage *tfe.TaskStage
	for _, stage := range taskStages.Items {
		if stage.Stage == tfe.PrePlan || stage.Stage == tfe.PostPlan {
			policyStage = stage
			break
		}
	}

	if policyStage == nil {
		log.Printf("[ERROR] No policy stage found in task stages")
		return nil, ErrNoPolicyCheck
	}

	// Read task stage with policy evaluations
	policyStageDetail, err := s.tfe.TaskStages.Read(ctx, policyStage.ID, &tfe.TaskStageReadOptions{
		Include: []tfe.TaskStageIncludeOpt{tfe.PolicyEvaluationsTaskResults},
	})
	if err != nil {
		return nil, fmt.Errorf("error reading task stage: %w", err)
	}

	result := &PolicyEvaluation{
		RunID:          run.ID,
		PolicyStageID:  policyStageDetail.ID,
		Status:         string(policyStageDetail.Status),
		RawAPIResponse: policyStageDetail,
	}

	// Aggregate counts from policy evaluations
	for _, policyEval := range policyStageDetail.PolicyEvaluations {
		if policyEval.ResultCount != nil {
			result.PassedCount += policyEval.ResultCount.Passed
			result.AdvisoryFailedCount += policyEval.ResultCount.AdvisoryFailed
			result.MandatoryFailedCount += policyEval.ResultCount.MandatoryFailed
			result.ErroredCount += policyEval.ResultCount.Errored
		}

		// Fetch detailed policy names for failed policies
		if policyEval.ResultCount != nil &&
			(policyEval.ResultCount.MandatoryFailed > 0 || policyEval.ResultCount.AdvisoryFailed > 0) {

			log.Printf("[INFO] Fetching policy set outcomes for evaluation %s", policyEval.ID)

			// Fetch policy set outcomes to get individual policy names
			outcomes, err := s.tfe.PolicySetOutcomes.List(ctx, policyEval.ID, nil)

			if err != nil {
				log.Printf("[WARN] Failed to fetch policy set outcomes for %s: %s", policyEval.ID, err)
				// Fall back to generic entry for mandatory failures
				if policyEval.ResultCount.MandatoryFailed > 0 {
					result.FailedPolicies = append(result.FailedPolicies, PolicyDetail{
						PolicyName:       fmt.Sprintf("policy-eval-%s", policyEval.ID),
						EnforcementLevel: EnforcementMandatory,
						Status:           PolicyStatusFailed,
						Description:      fmt.Sprintf("%d mandatory policies failed", policyEval.ResultCount.MandatoryFailed),
					})
				}
				continue
			}

			// Extract individual policy names from outcomes
			for _, policySetOutcome := range outcomes.Items {
				log.Printf("[DEBUG] Processing policy set: %s with %d outcomes", policySetOutcome.PolicySetName, len(policySetOutcome.Outcomes))

				for _, outcome := range policySetOutcome.Outcomes {
					// Only include failed policies
					if outcome.Status == "failed" {
						result.FailedPolicies = append(result.FailedPolicies, PolicyDetail{
							PolicyName:       outcome.PolicyName,
							EnforcementLevel: string(outcome.EnforcementLevel),
							Status:           outcome.Status,
							Description:      outcome.Description,
						})
						log.Printf("[DEBUG] Added failed policy: %s (%s)", outcome.PolicyName, outcome.EnforcementLevel)
					}
				}
			}
		}
	}

	result.TotalCount = result.PassedCount + result.AdvisoryFailedCount +
		result.MandatoryFailedCount + result.ErroredCount
	result.RequiresOverride = result.MandatoryFailedCount > 0

	log.Printf("[INFO] Policy evaluation retrieved: Total=%d, Passed=%d, Mandatory Failed=%d, Detailed Policies=%d",
		result.TotalCount, result.PassedCount, result.MandatoryFailedCount, len(result.FailedPolicies))

	return result, nil
}

// getPolicyFromPolicyCheck extracts policy evaluation from legacy API
func (s *policyService) getPolicyFromPolicyCheck(ctx context.Context, run *tfe.Run, check *tfe.PolicyCheck) *PolicyEvaluation {
	result := &PolicyEvaluation{
		RunID:          run.ID,
		PolicyCheckID:  check.ID,
		Status:         string(check.Status),
		RawAPIResponse: check,
	}

	// Use Result counts from PolicyCheck
	if check.Result != nil {
		result.PassedCount = check.Result.Passed
		result.AdvisoryFailedCount = check.Result.SoftFailed + check.Result.AdvisoryFailed
		result.MandatoryFailedCount = check.Result.HardFailed
		// Legacy API doesn't have explicit errored count; check status instead
		if check.Status == tfe.PolicyErrored {
			result.ErroredCount = 1
		}
		result.TotalCount = result.PassedCount + result.AdvisoryFailedCount + result.MandatoryFailedCount + result.ErroredCount
		result.RequiresOverride = result.MandatoryFailedCount > 0

		// Add generic failed policy entry if mandatory failures exist
		if result.MandatoryFailedCount > 0 {
			result.FailedPolicies = append(result.FailedPolicies, PolicyDetail{
				PolicyName:       fmt.Sprintf("policy-check-%s", check.ID),
				EnforcementLevel: EnforcementMandatory,
				Status:           PolicyStatusFailed,
				Description:      fmt.Sprintf("%d mandatory policies failed", result.MandatoryFailedCount),
			})
		}
	}

	log.Printf("[INFO] Policy check retrieved: Total=%d, Passed=%d, Mandatory Failed=%d",
		result.TotalCount, result.PassedCount, result.MandatoryFailedCount)

	return result
}
