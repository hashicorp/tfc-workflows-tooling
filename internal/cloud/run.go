// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/hashicorp/go-tfe"
	"github.com/sethvargo/go-retry"
)

const LogTimeout = time.Second * 10

var (
	ForceCancel              = tfe.RunStatus("force_canceled")
	PrePlanAwaitingDecision  = tfe.RunStatus("pre_apply_awaiting_decision")
	PostPlanAwaitingDecision = tfe.RunStatus("post_plan_awaiting_decision")
	PreApplyAwaitingDecision = tfe.RunStatus("pre_plan_awaiting_decision")
)

var DiscardNoopStatus = []tfe.RunStatus{
	tfe.RunErrored,
	tfe.RunCanceled,
	tfe.RunApplied,
	tfe.RunPlanned,
	tfe.RunPlannedAndFinished,
}

var CancelNoopStatus = []tfe.RunStatus{
	tfe.RunErrored,
	tfe.RunDiscarded,
	tfe.RunApplied,
	tfe.RunPlanned,
	tfe.RunPlannedAndFinished,
	tfe.RunPlannedAndSaved,
}

var NoopStatus = []tfe.RunStatus{
	tfe.RunErrored,
	tfe.RunCanceled,
	tfe.RunDiscarded,
	ForceCancel,
	PrePlanAwaitingDecision,
	PostPlanAwaitingDecision,
	PreApplyAwaitingDecision,
}

type CreateRunOptions struct {
	Organization           string
	Workspace              string
	ConfigurationVersionID string
	Message                string
	PlanOnly               bool
	IsDestroy              bool
	SavePlan               bool
	RunVariables           []*tfe.RunVariable
	TargetAddrs            []string
}

type ApplyRunOptions struct {
	RunID   string
	Comment string
}

type GetRunOptions struct {
	RunID string
}

type DiscardRunOptions struct {
	RunID   string
	Comment string
}

type CancelRunOptions struct {
	RunID       string
	Comment     string
	ForceCancel bool
}

type RunService interface {
	RunLink(context.Context, string, *tfe.Run) (string, error)
	GetRun(context.Context, GetRunOptions) (*tfe.Run, error)
	CreateRun(context.Context, CreateRunOptions) (*tfe.Run, error)
	ApplyRun(context.Context, ApplyRunOptions) (*tfe.Run, error)
	DiscardRun(context.Context, DiscardRunOptions) (*tfe.Run, error)
	CancelRun(context.Context, CancelRunOptions) (*tfe.Run, error)
	GetPlanLogs(context.Context, string) error
	GetApplyLogs(context.Context, string) error
	GetPolicyCheckLogs(context.Context, *tfe.Run) error
	LogCostEstimation(context.Context, *tfe.Run)
	LogTaskStage(context.Context, *tfe.Run, tfe.Stage) error
}

type runService struct {
	*cloudMeta
}

func (service *runService) RunLink(ctx context.Context, organization string, run *tfe.Run) (string, error) {
	wId := run.Workspace.ID
	tfWorkspace, err := service.tfe.Workspaces.ReadByID(ctx, wId)
	if err != nil {
		log.Printf("[ERROR] problem generating run link while fetching run by id: %s", wId)
		return "", err
	}
	url := service.tfe.BaseURL()
	link := fmt.Sprintf("%s://%s/app/%s/workspaces/%s/runs/%s", url.Scheme, url.Host, organization, tfWorkspace.Name, run.ID)
	service.writer.Output(fmt.Sprintf("View Run in Terraform Cloud: %s", link))

	return link, nil
}

func (service *runService) GetRun(ctx context.Context, options GetRunOptions) (*tfe.Run, error) {
	run, err := service.tfe.Runs.ReadWithOptions(ctx, options.RunID, &tfe.RunReadOptions{
		Include: []tfe.RunIncludeOpt{"cost_estimate", "plan"},
	})
	if err != nil {
		log.Printf("[ERROR] error reading run: %q error: %s", options.RunID, err)
		return nil, err
	}
	return run, nil
}

func (service *runService) CreateRun(ctx context.Context, options CreateRunOptions) (*tfe.Run, error) {
	var createOpts tfe.RunCreateOptions
	var cv *tfe.ConfigurationVersion
	// read workspace
	w, err := service.tfe.Workspaces.Read(ctx, options.Organization, options.Workspace)
	if err != nil {
		log.Printf("[ERROR] error reading workspace: %q organization: %q error: %s", options.Workspace, options.Organization, err)
		return nil, err
	}

	if w.Locked && !options.PlanOnly {
		return nil, errors.New("run has been specified as non-speculative and the workspace is currently locked")
	}

	if options.ConfigurationVersionID != "" {
		cv, err = service.tfe.ConfigurationVersions.Read(ctx, options.ConfigurationVersionID)
		if err != nil {
			log.Printf("[ERROR] error reading configuration version: %q error: %s", options.ConfigurationVersionID, err)
			return nil, err
		}
		createOpts.ConfigurationVersion = cv

		// if previously specified config version as speculative only and attempting to create a run
		// that is not, then return validation error
		if cv.Speculative && !options.PlanOnly {
			return nil, errors.New("configuration version has been specified as speculative and cannot be used in non-speculative runs")
		}
	}

	createOpts.Workspace = w
	createOpts.Message = &options.Message
	createOpts.PlanOnly = tfe.Bool(options.PlanOnly)
	createOpts.IsDestroy = tfe.Bool(options.IsDestroy)
	createOpts.SavePlan = tfe.Bool(options.SavePlan)
	createOpts.Variables = options.RunVariables
	createOpts.TargetAddrs = options.TargetAddrs

	// create the run
	run, err := service.tfe.Runs.Create(ctx, createOpts)

	if err != nil {
		log.Printf("[ERROR] error creating run in Terraform Cloud: %s", err)
		return nil, err
	}

	service.writer.Output(fmt.Sprintf("Created Run ID: %q", run.ID))

	costEstimateEnabled, policyChecksEnabled := hasCostEstimate(run), hasPolicyChecks(run)
	desiredStatus := getDesiredRunStatus(run, policyChecksEnabled, costEstimateEnabled)

	log.Printf("[DEBUG] PlanOnly: %t, AutoApply: %t, CostEstimation: %t, PolicyChecks: %t", run.PlanOnly, run.AutoApply, costEstimateEnabled, policyChecksEnabled)

	retryErr := retry.Do(ctx, defaultBackoff(), func(ctx context.Context) error {
		log.Printf("[DEBUG] Monitoring run status...")
		r, err := service.GetRun(ctx, GetRunOptions{
			RunID: run.ID,
		})

		// update run
		run = r

		if err != nil {
			return err
		}

		service.writer.Output(fmt.Sprintf("Run Status: %q", run.Status))

		done, err := isRunComplete(r, desiredStatus, NoopStatus)
		if err != nil {
			return err
		}

		if done {
			return nil
		}
		return retryableTimeoutError("create run ")
	})

	if retryErr != nil {
		return run, retryErr
	}

	return run, nil
}

func (service *runService) ApplyRun(ctx context.Context, options ApplyRunOptions) (*tfe.Run, error) {
	var applyRun *tfe.Run
	if err := service.tfe.Runs.Apply(ctx, options.RunID, tfe.RunApplyOptions{
		Comment: tfe.String(options.Comment),
	}); err != nil {
		log.Printf("[ERROR] error applying run: %q error: %s", options.RunID, err)
		return applyRun, err
	}

	if retryErr := retry.Do(ctx, defaultBackoff(), func(ctx context.Context) error {
		log.Printf("[DEBUG] Monitoring apply run status...")

		run, runErr := service.GetRun(ctx, GetRunOptions{
			RunID: options.RunID,
		})

		applyRun = run

		if runErr != nil {
			return runErr
		}

		service.writer.Output(fmt.Sprintf("Run Status: %q", run.Status))

		done, err := isRunComplete(run, []tfe.RunStatus{tfe.RunApplied}, NoopStatus)
		if err != nil {
			return err
		}

		if done {
			return nil
		}
		return retryableTimeoutError("apply run")
	}); retryErr != nil {
		return applyRun, retryErr
	}

	return applyRun, nil
}

func (service *runService) DiscardRun(ctx context.Context, options DiscardRunOptions) (*tfe.Run, error) {
	var discardRun *tfe.Run
	if err := service.tfe.Runs.Discard(ctx, options.RunID, tfe.RunDiscardOptions{
		Comment: &options.Comment,
	}); err != nil {
		log.Printf("[ERROR] error discarding run: %q error: %s", options.RunID, err.Error())
		return discardRun, err
	}

	if retryErr := retry.Do(ctx, defaultBackoff(), func(context context.Context) error {
		log.Printf("[DEBUG] Monitoring discard run status...")
		run, runErr := service.GetRun(ctx, GetRunOptions{
			RunID: options.RunID,
		})

		discardRun = run

		if runErr != nil {
			return runErr
		}

		service.writer.Output(fmt.Sprintf("Run Status: %q", run.Status))

		done, err := isRunComplete(run, []tfe.RunStatus{tfe.RunDiscarded}, DiscardNoopStatus)
		if err != nil {
			return err
		}

		if done {
			return nil
		}
		return retryableTimeoutError("discard run")
	}); retryErr != nil {
		return discardRun, retryErr
	}

	return discardRun, nil
}

func (service *runService) CancelRun(ctx context.Context, options CancelRunOptions) (*tfe.Run, error) {
	var cancelRun *tfe.Run
	var err error
	if options.ForceCancel {
		err = service.tfe.Runs.ForceCancel(ctx, options.RunID, tfe.RunForceCancelOptions{
			Comment: &options.Comment,
		})
	} else {
		err = service.tfe.Runs.Cancel(ctx, options.RunID, tfe.RunCancelOptions{
			Comment: &options.Comment,
		})
	}

	if err != nil {
		log.Printf("[ERROR] error canceling run: %q, with: %s", options.RunID, err.Error())
		return cancelRun, err
	}

	retryErr := retry.Do(ctx, defaultBackoff(), func(context context.Context) error {
		log.Printf("[DEBUG] Monitoring cancel run status...")
		run, runErr := service.GetRun(ctx, GetRunOptions{
			RunID: options.RunID,
		})

		cancelRun = run

		if runErr != nil {
			return runErr
		}

		service.writer.Output(fmt.Sprintf("Run Status: %q", run.Status))

		done, err := isRunComplete(run, []tfe.RunStatus{tfe.RunCanceled}, CancelNoopStatus)
		if err != nil {
			return err
		}

		if done {
			return nil
		}
		return retryableTimeoutError("cancel run")
	})
	if retryErr != nil {
		return cancelRun, retryErr
	}

	return cancelRun, nil
}

func (service *runService) GetPlanLogs(ctx context.Context, planID string) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, LogTimeout)
	defer cancel()

	var err error
	var logReader io.Reader
	logReader, err = service.tfe.Plans.Logs(ctxTimeout, planID)
	if err != nil {
		return err
	}

	service.writer.Output(fmt.Sprintf("-------------- %s --------------", "Plan Log"))
	err = outputRunLogLines(logReader, service.writer)
	if err != nil {
		return err
	}
	fmt.Println()
	return nil
}

func (service *runService) GetApplyLogs(ctx context.Context, applyID string) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, LogTimeout)
	defer cancel()

	var err error
	var logReader io.Reader
	logReader, err = service.tfe.Applies.Logs(ctxTimeout, applyID)
	if err != nil {
		return err
	}

	service.writer.Output(fmt.Sprintf("-------------- %s --------------", "Apply Log"))
	err = outputRunLogLines(logReader, service.writer)
	if err != nil {
		return err
	}
	fmt.Println()
	return nil
}

func (s *runService) GetPolicyCheckLogs(ctx context.Context, run *tfe.Run) error {
	if !(len(run.PolicyChecks) > 0) {
		return nil
	}

	policyChecks, err := s.tfe.PolicyChecks.List(ctx, run.ID, &tfe.PolicyCheckListOptions{})
	if err != nil {
		return err
	}

	logStart := true
	fmt.Println()
	for _, pcheck := range policyChecks.Items {
		ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		// if no work was done, skip
		if pcheck.Status == tfe.PolicyPending || pcheck.Status == tfe.PolicyUnreachable {
			continue
		}

		var err error
		var logReader io.Reader
		logReader, err = s.tfe.PolicyChecks.Logs(ctxTimeout, pcheck.ID)
		if err != nil {
			return err
		}

		// only log for first sentinel policy
		if logStart {
			s.writer.Output(fmt.Sprintf("-------------- %s --------------", "Sentinel Policy Checks"))
			logStart = false
		}

		err = outputRunLogLines(logReader, s.writer)
		if err != nil {
			return err
		}
		fmt.Println()
	}

	return nil
}

func (s *runService) LogTaskStage(ctx context.Context, run *tfe.Run, stage tfe.Stage) error {
	taskStages, err := s.tfe.TaskStages.List(ctx, run.ID, &tfe.TaskStageListOptions{})
	if err != nil {
		return err
	}
	if !(len(taskStages.Items) > 0) {
		return nil
	}

	labelMap := map[string]string{
		"post_plan": "Post Plan",
		"pre_plan":  "Pre Plan",
		"pre_apply": "Pre Apply",
	}

	fmt.Println()
	for _, task := range taskStages.Items {
		if task.Stage == stage {
			s.writer.Output(fmt.Sprintf("-------------- %s --------------", labelMap[string(stage)]))
			s.writer.Output(fmt.Sprintf("TaskStage (%s), Status: '%s', Stage: '%s'", task.ID, task.Status, task.Stage))
			for _, taskResult := range task.TaskResults {
				taskResult, resErr := s.tfe.TaskResults.Read(ctx, taskResult.ID)
				if resErr != nil {
					return fmt.Errorf("error reading results for task results: %s", resErr.Error())
				}
				s.writer.Output(fmt.Sprintf("- TaskResult (%s), Name: '%s', Status: '%s', EnforcementLevel: '%s', Message: '%s'", taskResult.ID, taskResult.TaskName, taskResult.Status, taskResult.WorkspaceTaskEnforcementLevel, taskResult.Message))
			}
			evaluations, pErr := s.tfe.PolicyEvaluations.List(ctx, task.ID, &tfe.PolicyEvaluationListOptions{})
			if pErr != nil {
				return fmt.Errorf("error reading results for policy evaluations: %s", pErr.Error())
			}
			for _, p := range evaluations.Items {
				s.writer.Output(fmt.Sprintf("- PolicyEvalutation (%s), Status: '%s', PolicyKind: '%s'", p.ID, p.Status, p.PolicyKind))
				s.writer.Output(fmt.Sprintf("  Passed: (%d), AdvisoryFailed: (%d), MandatoryFailed: (%d), Failed: (%d)", p.ResultCount.Passed, p.ResultCount.AdvisoryFailed, p.ResultCount.MandatoryFailed, p.ResultCount.Errored))
			}
			fmt.Println()
		}
	}
	return nil
}

func (s *runService) LogCostEstimation(ctx context.Context, run *tfe.Run) {
	if run.CostEstimate == nil || run.CostEstimate.Status == tfe.CostEstimateStatus("unreachable") || run.CostEstimate.Status == tfe.CostEstimatePending {
		return
	}

	s.writer.Output(fmt.Sprintf("-------------- CostEstimation (%s) --------------", run.CostEstimate.ID))
	s.writer.Output(fmt.Sprintf("Status: %q, ErrorMessage: %q", run.CostEstimate.Status, run.CostEstimate.ErrorMessage))
	s.writer.Output(fmt.Sprintf("PriorMonthlyCost: (%s), ProposedMonthlyCost: (%s), Delta: (%s)", run.CostEstimate.PriorMonthlyCost, run.CostEstimate.ProposedMonthlyCost, run.CostEstimate.DeltaMonthlyCost))
	fmt.Println()
}

func outputRunLogLines(logs io.Reader, writer Writer) error {
	var err error
	reader := bufio.NewReaderSize(logs, 64*1024)
	for next := true; next; {
		var l, line []byte

		for isPrefix := true; isPrefix; {
			l, isPrefix, err = reader.ReadLine()
			if err != nil {
				if err != io.EOF {
					return err
				}
				next = false
			}
			line = append(line, l...)
		}

		if next || len(line) > 0 {
			writer.Output(string(line))
		}
	}
	return nil
}

func NewRunService(meta *cloudMeta) RunService {
	return &runService{meta}
}

func getDesiredRunStatus(run *tfe.Run, policyChecksEnabled bool, costEstimateEnabled bool) []tfe.RunStatus {
	// shared desired status across all runs
	desiredStatus := []tfe.RunStatus{
		tfe.RunPolicySoftFailed,
		tfe.RunPlannedAndFinished,
		tfe.RunApplied,
	}

	if run.SavePlan {
		desiredStatus = append(desiredStatus, tfe.RunPlannedAndSaved)
	}

	// when plan_only run
	if run.PlanOnly {
		// plan only runs will result in default desired slice
		// most likely planned_and_finished or policy_soft_failed
		return desiredStatus
	}

	// when auto_apply run
	if run.AutoApply {
		if policyChecksEnabled {
			// policy override requires human approval before proceeding, run has reached no-op
			desiredStatus = append(desiredStatus, tfe.RunPolicyOverride)
		}
		return desiredStatus
	}

	// when applyable/confirmable run
	// determine which various run status it can end with
	if costEstimateEnabled && !policyChecksEnabled {
		// cost_estimation executes prior to sentinel policies
		// if we expect `"cost_estimated"` as a final step, the cmd will eject too early
		desiredStatus = append(desiredStatus, tfe.RunCostEstimated)
	} else if policyChecksEnabled {
		// account for `"policy_checked"` & `"policy_override"` if policy checks are enabled for the run
		desiredStatus = append(desiredStatus, tfe.RunPolicyChecked, tfe.RunPolicyOverride)
	} else {
		// applyable/confirmable run has no cost estimation nor sentinel policies
		// run task stages are accounted for in the Noop RunStatus slice
		desiredStatus = append(desiredStatus, tfe.RunPlanned)
	}

	return desiredStatus
}

func isRunComplete(run *tfe.Run, desiredStatus []tfe.RunStatus, noopStatus []tfe.RunStatus) (done bool, err error) {
	for _, s := range desiredStatus {
		if run.Status == s {
			return true, nil
		}
	}
	for _, v := range noopStatus {
		// we've reached non operable state, return error
		if run.Status == v {
			return true, fmt.Errorf("run has ended with: '%s' status", run.Status)
		}
	}
	return false, nil
}

func hasCostEstimate(run *tfe.Run) bool {
	enabled := false
	costEstimate := run.CostEstimate
	if costEstimate != nil {
		enabled = costEstimate.ID != ""
	}
	return enabled
}

func hasPolicyChecks(run *tfe.Run) bool {
	enabled := false
	if len(run.PolicyChecks) > 0 {
		enabled = true
	}
	return enabled
}
