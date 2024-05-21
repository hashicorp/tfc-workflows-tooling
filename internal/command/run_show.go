// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/tfci/internal/cloud"
)

type ShowRunCommand struct {
	*Meta

	RunID string
}

func (c *ShowRunCommand) flags() *flag.FlagSet {
	f := c.flagSet("run show")
	f.StringVar(&c.RunID, "run", "", "Existing HCP Terraform Run ID to show.")

	return f
}

func (c *ShowRunCommand) Run(args []string) int {
	if err := c.setupCmd(args, c.flags()); err != nil {
		return 1
	}

	if c.RunID == "" {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.writer.ErrorResult("showing a run requires a valid run id")
		return 1
	}

	// fetch run
	run, err := c.cloud.GetRun(c.appCtx, cloud.GetRunOptions{
		RunID: c.RunID,
	})

	if err != nil {
		status := c.resolveStatus(err)
		c.addOutput("status", string(status))
		c.addRunDetails(run)
		c.writer.ErrorResult(fmt.Sprintf("error showing run, '%s' in HCP Terraform: %s", c.RunID, err.Error()))
		c.writer.OutputResult(c.closeOutput())
		return 1
	}

	c.addOutput("status", string(Success))
	c.addRunDetails(run)
	c.writer.OutputResult(c.closeOutput())
	return 0
}

func (c *ShowRunCommand) addRunDetails(run *tfe.Run) {
	if run == nil {
		return
	}

	runLink, _ := c.cloud.RunLink(c.appCtx, c.organization, run)
	if runLink != "" {
		c.addOutput("run_link", runLink)
	}
	c.addOutput("run_id", run.ID)
	c.addOutput("run_status", string(run.Status))
	c.addOutput("run_message", run.Message)
	c.addOutput("plan_id", run.Plan.ID)
	c.addOutput("plan_status", string(run.Plan.Status))
	c.addOutput("configuration_version_id", run.ConfigurationVersion.ID)

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

func (c *ShowRunCommand) Help() string {
	helpText := `
Usage: tfci [global options] run show [options]

	Returns run details for the provided HCP Terraform run ID.

Global Options:

	-hostname       The hostname of a Terraform Enterprise installation, if using Terraform Enterprise. Defaults to "app.terraform.io".

	-token          The token used to authenticate with HCP Terraform. Defaults to reading "TF_API_TOKEN" environment variable.

	-organization   HCP Terraform Organization Name.

Options:

	-run            Existing HCP Terraform Run ID to show.
	`
	return strings.TrimSpace(helpText)
}

func (c *ShowRunCommand) Synopsis() string {
	return "Returns run details for the provided HCP Terraform run ID"
}
