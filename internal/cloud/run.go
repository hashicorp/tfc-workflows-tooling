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
	tfe *tfe.Client
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
	fmt.Printf("View Run in Terraform Cloud: %s\n", link)

	return link, nil
}

func (service *runService) GetRun(ctx context.Context, options GetRunOptions) (*tfe.Run, error) {
	run, err := service.tfe.Runs.ReadWithOptions(ctx, options.RunID, &tfe.RunReadOptions{
		Include: []tfe.RunIncludeOpt{"cost_estimate", "plan"},
	})
	if err != nil {
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
		return nil, err
	}

	if w.Locked && !options.PlanOnly {
		return nil, errors.New("run has been specified as non-speculative and the workspace is currently locked")
	}

	if options.ConfigurationVersionID != "" {
		cv, err = service.tfe.ConfigurationVersions.Read(ctx, options.ConfigurationVersionID)
		if err != nil {
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
	createOpts.Variables = options.RunVariables

	// create the run
	run, err := service.tfe.Runs.Create(ctx, createOpts)

	if err != nil {
		return nil, err
	}

	fmt.Printf("Created Run ID: %s\n", run.ID)

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

		costEstimateEnabled, policyChecksEnabled := hasCostEstimate(r), hasPolicyChecks(r)

		fmt.Printf("Run Status: '%s'\n", run.Status)

		log.Printf("[DEBUG] PlanOnly: %t, CostEstimation: %t, PolicyChecks: %t", r.PlanOnly, costEstimateEnabled, policyChecksEnabled)

		desiredStatus := []tfe.RunStatus{
			tfe.RunPolicySoftFailed,
			tfe.RunPlannedAndFinished,
			tfe.RunApplied,
		}

		if !r.PlanOnly {
			if costEstimateEnabled && !policyChecksEnabled {
				desiredStatus = append(desiredStatus, tfe.RunCostEstimated)
			} else if policyChecksEnabled {
				desiredStatus = append(desiredStatus, tfe.RunPolicyChecked, tfe.RunPolicyOverride)
			} else {
				desiredStatus = append(desiredStatus, tfe.RunPlanned)
			}
		}

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

		fmt.Printf("Run Status: '%s'\n", run.Status)

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

		fmt.Printf("Run Status: '%s'\n", run.Status)

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

		fmt.Printf("Run Status: '%s'\n", run.Status)

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

	fmt.Printf("\n-------------- %s --------------\n", "Plan Log")
	err = outputRunLogLines(logReader)
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

	fmt.Printf("\n-------------- %s --------------\n", "Apply Log")
	err = outputRunLogLines(logReader)
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
			fmt.Println("-------------- Sentinel Policy Checks --------------")
			logStart = false
		}

		err = outputRunLogLines(logReader)
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
			fmt.Printf("-------------- %s --------------\n", labelMap[string(stage)])
			fmt.Printf("TaskStage (%s), Status: '%s', Stage: '%s'\n", task.ID, task.Status, task.Stage)
			for _, taskResult := range task.TaskResults {
				taskResult, resErr := s.tfe.TaskResults.Read(ctx, taskResult.ID)
				if resErr != nil {
					return fmt.Errorf("error reading results for task results: %s", resErr.Error())
				}
				fmt.Printf("- TaskResult (%s), Name: '%s', Status: '%s', EnforcementLevel: '%s', Message: '%s'\n", taskResult.ID, taskResult.TaskName, taskResult.Status, taskResult.WorkspaceTaskEnforcementLevel, taskResult.Message)
			}
			evaluations, pErr := s.tfe.PolicyEvaluations.List(ctx, task.ID, &tfe.PolicyEvaluationListOptions{})
			if pErr != nil {
				return fmt.Errorf("error reading results for policy evaluations: %s", pErr.Error())
			}
			for _, p := range evaluations.Items {
				fmt.Printf("- PolicyEvalutation (%s), Status: '%s', PolicyKind: '%s'\n", p.ID, p.Status, p.PolicyKind)
				fmt.Printf("  Passed: (%d), AdvisoryFailed: (%d), MandatoryFailed: (%d), Failed: (%d)\n", p.ResultCount.Passed, p.ResultCount.AdvisoryFailed, p.ResultCount.MandatoryFailed, p.ResultCount.Errored)
			}
			fmt.Println()
		}
	}
	return nil
}

func (s *runService) LogCostEstimation(ctx context.Context, run *tfe.Run) {
	checkStatus := func(s tfe.CostEstimateStatus) bool {
		for _, status := range []tfe.CostEstimateStatus{tfe.CostEstimateStatus("unreachable"), tfe.CostEstimatePending} {
			if s == status {
				return false
			}
		}
		return true
	}

	if run.CostEstimate != nil && checkStatus(run.CostEstimate.Status) {
		fmt.Printf("\n-------------- CostEstimation (%s) --------------\n", run.CostEstimate.ID)
		fmt.Printf("Status: '%s', ErrorMessage: '%s'\n", run.CostEstimate.Status, run.CostEstimate.ErrorMessage)
		fmt.Printf("PriorMonthlyCost: (%s), ProposedMonthlyCost: (%s), Delta: (%s)\n", run.CostEstimate.PriorMonthlyCost, run.CostEstimate.ProposedMonthlyCost, run.CostEstimate.DeltaMonthlyCost)
		fmt.Println()
	}
}

func outputRunLogLines(logs io.Reader) error {
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
			fmt.Println(string(line))
		}
	}
	return nil
}

func NewRunService(tfe *tfe.Client) RunService {
	return &runService{tfe}
}

type CreateRunOptions struct {
	Organization           string
	Workspace              string
	ConfigurationVersionID string
	Message                string
	PlanOnly               bool
	RunVariables           []*tfe.RunVariable
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
