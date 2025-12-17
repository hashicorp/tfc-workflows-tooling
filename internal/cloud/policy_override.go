// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-tfe"
	"github.com/sethvargo/go-retry"
)

// OverridePolicy applies a policy override with justification
func (s *policyService) OverridePolicy(ctx context.Context, options OverridePolicyOptions) (*PolicyOverride, error) {
	// Validate options
	if err := options.Validate(); err != nil {
		return nil, err
	}

	log.Printf("[INFO] Applying policy override for run %s", options.RunID)

	// Validate run status and override eligibility
	run, err := s.validateOverrideEligibility(ctx, options.RunID)
	if err != nil {
		return nil, err
	}

	// Detect API format and apply override
	result := &PolicyOverride{
		RunID:         run.ID,
		Justification: options.Justification,
		InitialStatus: string(run.Status),
		Timestamp:     time.Now().UTC(),
	}

	// Try modern API first
	taskStages, err := s.tfe.TaskStages.List(ctx, run.ID, &tfe.TaskStageListOptions{})
	if err == nil && taskStages != nil && len(taskStages.Items) > 0 {
		log.Printf("[DEBUG] Using modern API (task-stages) for policy override")
		return s.overrideViaTaskStage(ctx, run, result, taskStages)
	}

	// Fall back to legacy API
	log.Printf("[DEBUG] Using legacy API (policy-checks) for policy override")

	// Read run again to get policy checks relationship
	run, err = s.tfe.Runs.ReadWithOptions(ctx, options.RunID, &tfe.RunReadOptions{
		Include: []tfe.RunIncludeOpt{RunPolicyChecks},
	})
	if err != nil {
		return nil, fmt.Errorf("error reading run for policy checks: %w", err)
	}

	if len(run.PolicyChecks) == 0 {
		return nil, ErrNoPolicyCheck
	}

	return s.overrideViaPolicyCheck(ctx, run, result)
}

// validateOverrideEligibility checks if a run can be overridden
func (s *policyService) validateOverrideEligibility(ctx context.Context, runID string) (*tfe.Run, error) {
	run, err := s.tfe.Runs.Read(ctx, runID)
	if err != nil {
		log.Printf("[ERROR] Failed to read run %s: %s", runID, err)
		return nil, fmt.Errorf("error reading run: %w", err)
	}

	if run.Status != PostPlanAwaitingDecision {
		log.Printf("[ERROR] Cannot override run %s: status is %s, expected post_plan_awaiting_decision", runID, run.Status)
		return nil, fmt.Errorf("%w: run status is %s, expected post_plan_awaiting_decision", ErrInvalidRunStatus, run.Status)
	}

	return run, nil
}

// overrideViaTaskStage applies override using modern API
func (s *policyService) overrideViaTaskStage(ctx context.Context, run *tfe.Run, result *PolicyOverride, taskStages *tfe.TaskStageList) (*PolicyOverride, error) {
	var policyStage *tfe.TaskStage
	for _, stage := range taskStages.Items {
		if stage.Stage == tfe.PrePlan || stage.Stage == tfe.PostPlan {
			policyStage = stage
			break
		}
	}

	if policyStage == nil {
		return nil, ErrNoPolicyCheck
	}

	result.PolicyStageID = policyStage.ID

	// Apply override
	log.Printf("[DEBUG] Applying override to task stage %s", policyStage.ID)
	_, overrideErr := s.tfe.TaskStages.Override(ctx, policyStage.ID, tfe.TaskStageOverrideOptions{
		Comment: &result.Justification,
	})
	if overrideErr != nil {
		log.Printf("[ERROR] Failed to override task stage: %s", overrideErr)
		return nil, fmt.Errorf("error overriding task stage: %w", overrideErr)
	}

	// Add justification comment to run
	if err := s.addJustificationComment(ctx, run.ID, result.Justification); err != nil {
		log.Printf("[WARN] Failed to add comment to run: %s", err)
	}

	// Poll for status change
	finalRun, err := s.waitForOverrideCompletion(ctx, run.ID)
	if err != nil {
		return nil, err
	}

	result.FinalStatus = string(finalRun.Status)
	result.OverrideComplete = true

	log.Printf("[INFO] Policy override completed: %s → %s", result.InitialStatus, result.FinalStatus)

	return result, nil
}

// overrideViaPolicyCheck applies override using legacy API
func (s *policyService) overrideViaPolicyCheck(ctx context.Context, run *tfe.Run, result *PolicyOverride) (*PolicyOverride, error) {
	// Guard against empty policy checks
	if len(run.PolicyChecks) == 0 {
		return nil, ErrNoPolicyCheck
	}

	// Use the first policy check
	policyCheck := run.PolicyChecks[0]
	result.PolicyCheckID = policyCheck.ID

	// Apply override (legacy API is synchronous)
	log.Printf("[DEBUG] Applying override to policy check %s", policyCheck.ID)
	_, err := s.tfe.PolicyChecks.Override(ctx, policyCheck.ID)
	if err != nil {
		log.Printf("[ERROR] Failed to override policy check: %s", err)
		return nil, fmt.Errorf("error overriding policy check: %w", err)
	}

	// Add justification comment to run
	if err := s.addJustificationComment(ctx, run.ID, result.Justification); err != nil {
		log.Printf("[WARN] Failed to add comment to run: %s", err)
	}

	// Poll for status change
	finalRun, err := s.waitForOverrideCompletion(ctx, run.ID)
	if err != nil {
		return nil, err
	}

	result.FinalStatus = string(finalRun.Status)
	result.OverrideComplete = true

	log.Printf("[INFO] Policy override completed: %s → %s", result.InitialStatus, result.FinalStatus)

	return result, nil
}

// addJustificationComment adds a comment to the run explaining the override
func (s *policyService) addJustificationComment(ctx context.Context, runID, justification string) error {
	_, err := s.tfe.Comments.Create(ctx, runID, tfe.CommentCreateOptions{
		Body: fmt.Sprintf("Policy Override: %s", justification),
	})
	return err
}

// waitForOverrideCompletion polls until override status transition completes
func (s *policyService) waitForOverrideCompletion(ctx context.Context, runID string) (*tfe.Run, error) {
	log.Printf("[DEBUG] Waiting for override to complete for run %s", runID)

	backoff := policyWaitBackoffStrategy()
	var finalRun *tfe.Run

	err := retry.Do(ctx, backoff, func(ctx context.Context) error {
		var err error
		finalRun, err = s.tfe.Runs.Read(ctx, runID)
		if err != nil {
			return fmt.Errorf("error reading run: %w", err)
		}

		log.Printf("[DEBUG] Polling run status after override: %s", finalRun.Status)

		// Check for expected post-override states
		switch finalRun.Status {
		case tfe.RunPolicyOverride, tfe.RunPostPlanCompleted, tfe.RunApplyQueued:
			// Override completed successfully
			return nil
		case tfe.RunDiscarded:
			return fmt.Errorf("run was discarded during override")
		case tfe.RunCanceled, tfe.RunErrored:
			return fmt.Errorf("run entered terminal state %s during override", finalRun.Status)
		case PostPlanAwaitingDecision:
			// Still waiting for override to take effect
			return retry.RetryableError(fmt.Errorf("override still processing"))
		default:
			// Unexpected status, but keep retrying
			log.Printf("[WARN] Unexpected run status during override: %s", finalRun.Status)
			return retry.RetryableError(fmt.Errorf("unexpected status: %s", finalRun.Status))
		}
	})

	if err != nil {
		return nil, err
	}

	return finalRun, nil
}
