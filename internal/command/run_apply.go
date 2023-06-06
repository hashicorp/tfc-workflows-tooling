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

type ApplyRunCommand struct {
	*Meta

	RunID   string
	Comment string
}

func (c *ApplyRunCommand) flags() *flag.FlagSet {
	f := c.flagSet("run apply")
	f.StringVar(&c.RunID, "run", "", "Existing Terraform Cloud Run ID to Apply.")
	f.StringVar(&c.Comment, "comment", "", "An optional comment about the run.")

	return f
}

func (c *ApplyRunCommand) Run(args []string) int {
	flags := c.flags()
	if err := flags.Parse(args); err != nil {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.Ui.Error(fmt.Sprintf("error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	if c.RunID == "" {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.Ui.Error("applying a run requires a valid run id")
		return 1
	}

	// fetch existing run details
	run, runErr := c.cloud.GetRun(c.Context, cloud.GetRunOptions{
		RunID: c.RunID,
	})

	if runErr != nil {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.Ui.Error(fmt.Sprintf("unable to read run: %s with: %s", c.RunID, runErr.Error()))
		return 1
	}

	// check if run can be applied at this moment
	if !run.Actions.IsConfirmable {
		if run.Status == tfe.RunPlannedAndFinished {
			c.addOutput("status", string(Noop))
			c.addRunDetails(run)
			c.Ui.Error(fmt.Sprintf("run %s, is planned and finished. There is nothing to do.", c.RunID))
			c.Ui.Output(c.closeOutput())
			return 0
		}
		c.addOutput("status", string(Error))
		c.addRunDetails(run)
		c.Ui.Error(fmt.Sprintf("run %s, cannot be applied", c.RunID))
		c.Ui.Output(c.closeOutput())
		return 1
	}

	latestRun, applyError := c.cloud.ApplyRun(c.Context, cloud.ApplyRunOptions{
		RunID:   c.RunID,
		Comment: c.Comment,
	})
	if latestRun != nil {
		run = latestRun
		c.readApplyLogs(run)
	}

	if applyError != nil {
		status := c.resolveStatus(applyError)
		c.addOutput("status", string(status))
		c.addRunDetails(run)
		c.Ui.Error(fmt.Sprintf("error applying run, '%s' in Terraform Cloud: %s", c.RunID, applyError.Error()))
		c.Ui.Output(c.closeOutput())
		return 1
	}

	c.addOutput("status", string(Success))
	c.addRunDetails(run)
	c.Ui.Output(c.closeOutput())
	return 0
}

func (c *ApplyRunCommand) addRunDetails(run *tfe.Run) {
	if run == nil {
		return
	}
	link, _ := c.cloud.RunLink(c.Context, c.Organization, run)
	if link != "" {
		c.addOutput("run_link", link)
	}
	c.addOutput("run_id", run.ID)
	c.addOutput("run_status", string(run.Status))
}

func (c *ApplyRunCommand) readApplyLogs(run *tfe.Run) {
	// pre-apply task stage
	c.cloud.LogTaskStage(c.Context, run, tfe.PreApply)
	// apply logs
	if logErr := c.cloud.GetApplyLogs(c.Context, run.Apply.ID); logErr != nil {
		c.Ui.Error(fmt.Sprintf("failed to read apply logs: %s", logErr.Error()))
	}
}

func (c *ApplyRunCommand) Help() string {
	helpText := `
Usage: tfci [global options] run apply [options]

	Applies a run that is paused waiting for confirmation after a plan.

Global Options:

	-hostname       The hostname of a Terraform Enterprise installation, if using Terraform Enterprise. Defaults to "app.terraform.io".

	-token          The token used to authenticate with Terraform Cloud. Defaults to reading "TF_API_TOKEN" environment variable.

	-organization   Terraform Cloud Organization Name.

Options:

	-run         Existing Terraform Cloud Run ID to Apply.

	-comment     An optional comment about the run.
	`
	return strings.TrimSpace(helpText)
}

func (c *ApplyRunCommand) Synopsis() string {
	return "Applies a run that is paused waiting for confirmation after a plan"
}
