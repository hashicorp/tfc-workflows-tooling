// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/tfci/internal/cloud"
)

type CreateRunCommand struct {
	*Meta

	Workspace              string
	ConfigurationVersionID string
	Message                string

	PlanOnly bool
}

func (c *CreateRunCommand) flags() *flag.FlagSet {
	f := c.flagSet("run create")
	f.StringVar(&c.Workspace, "workspace", "", "The name of the Terraform Cloud Workspace.")
	f.StringVar(&c.ConfigurationVersionID, "configuration_version", "", "The Configuration Version ID to use for this run.")
	f.StringVar(&c.Message, "message", "", "Specifies the message to be associated with this run. A default message will be set.")
	f.BoolVar(&c.PlanOnly, "plan-only", false, "Specifies if this is a Terraform Cloud speculative, plan-only run that cannot be applied.")

	return f
}

func (c *CreateRunCommand) Run(args []string) int {
	flags := c.flags()
	if err := flags.Parse(args); err != nil {
		c.emitFlagOptions()
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.writer.ErrorResult(fmt.Sprintf("error parsing command-line flags: %s\n", err.Error()))
		return 1
	}
	c.emitFlagOptions()

	runVars := collectVariables()

	// default formatted message for run, include vcs ci runner information
	if c.Message == "" {
		c.Message = c.defaultRunMessage()
	}

	run, runError := c.cloud.CreateRun(c.appCtx, cloud.CreateRunOptions{
		Organization:           c.organization,
		Workspace:              c.Workspace,
		ConfigurationVersionID: c.ConfigurationVersionID,
		Message:                c.Message,
		PlanOnly:               c.PlanOnly,
		RunVariables:           runVars,
	})
	if run != nil {
		c.readPlanLogs(run)
	}

	if runError != nil {
		status := c.resolveStatus(runError)
		errMsg := fmt.Sprintf("error while creating run in Terraform Cloud: %s", runError.Error())
		c.addOutput("status", string(status))
		c.addRunDetails(run)
		c.writer.ErrorResult(errMsg)
		c.writer.OutputResult(c.closeOutput())
		return 1
	}

	c.addOutput("status", string(Success))
	c.addRunDetails(run)
	c.writer.OutputResult(c.closeOutput())
	return 0
}

func (c *CreateRunCommand) addRunDetails(run *tfe.Run) {
	if run == nil {
		log.Printf("[ERROR] run is not detected")
		return
	}
	runLink, _ := c.cloud.RunService.RunLink(c.appCtx, c.organization, run)
	if runLink != "" {
		c.addOutput("run_link", runLink)
	}
	c.addOutput("run_id", run.ID)
	c.addOutput("run_status", string(run.Status))
	c.addOutput("run_message", run.Message)
	c.addOutput("plan_id", run.Plan.ID)
	c.addOutput("plan_status", string(run.Plan.Status))
	c.addOutput("configuration_version_id", run.ConfigurationVersion.ID)

	// add cost estimation info if enabled on run
	if run.CostEstimate != nil {
		c.addOutput("cost_estimation_id", run.CostEstimate.ID)
		c.addOutput("cost_estimation_status", string(run.CostEstimate.Status))
		if run.CostEstimate.ErrorMessage != "" {
			c.writer.ErrorResult(fmt.Sprintf("Cost Estimation errored: %s", run.CostEstimate.ErrorMessage))
		}
	}

	c.addOutputWithOpts("payload", run, &outputOpts{
		stdOut:      false,
		multiLine:   true,
		platformOut: true,
	})
}

func (c *CreateRunCommand) readPlanLogs(run *tfe.Run) {
	// Pre Plan task stages
	c.cloud.LogTaskStage(c.appCtx, run, tfe.PrePlan)
	// Plan
	if pLogErr := c.cloud.GetPlanLogs(c.appCtx, run.Plan.ID); pLogErr != nil {
		c.writer.ErrorResult(fmt.Sprintf("failed to read plan logs: %s", pLogErr.Error()))
	}
	// Post Plan task stages
	c.cloud.LogTaskStage(c.appCtx, run, tfe.PostPlan)
	// cost estimation
	c.cloud.LogCostEstimation(c.appCtx, run)
	// sentinel policies
	if policyLogErr := c.cloud.GetPolicyCheckLogs(c.appCtx, run); policyLogErr != nil {
		c.writer.ErrorResult(fmt.Sprintf("failed to read policy check logs: %s", policyLogErr.Error()))
	}
}

func (c *CreateRunCommand) defaultRunMessage() string {
	if c.env.Context != nil {
		return fmt.Sprintf("Triggered from Terraform Cloud CI by Author (%s) for SHA (%s)", c.env.Context.Author(), c.env.Context.SHAShort())
	}
	return `Triggered from Terraform Cloud CI`
}

func (c *CreateRunCommand) Help() string {
	helpText := `
Usage: tfci [global options] run create [options]

	Performs a new plan run in Terraform Cloud, using a configuration version and the workspace's current variables.

Global Options:

	-hostname       The hostname of a Terraform Enterprise installation, if using Terraform Enterprise. Defaults to "app.terraform.io".

	-token          The token used to authenticate with Terraform Cloud. Defaults to reading "TF_API_TOKEN" environment variable.

	-organization   Terraform Cloud Organization Name.

Options:

	-workspace              The name of the Terraform Cloud Workspace.

	-configuration_version  The Configuration Version ID to use for this run.

	-message                Specifies the message to be associated with this run. A default message will be set.

	-plan-only              Specifies if this is a Terraform Cloud speculative, plan-only run that cannot be applied.
	`
	return strings.TrimSpace(helpText)
}

func (c *CreateRunCommand) Synopsis() string {
	return "Performs a new plan run in Terraform Cloud, using a configuration version and the workspace's current variables"
}
