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

type CancelRunCommand struct {
	*Meta

	RunID       string
	Comment     string
	ForceCancel bool
}

func (c *CancelRunCommand) flags() *flag.FlagSet {
	f := c.flagSet("run cancel")
	f.StringVar(&c.RunID, "run", "", "Existing Terraform Cloud Run ID to Discard")
	f.StringVar(&c.Comment, "comment", "", "An optional comment about the run.")
	f.BoolVar(&c.ForceCancel, "force-cancel", false, "Ends the run immediately.")

	return f
}

func (c *CancelRunCommand) Run(args []string) int {
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
		c.Ui.Error("cancelling a run requires a run id")
		return 1
	}

	// fetch existing run details
	run, runErr := c.cloud.GetRun(c.Context, cloud.GetRunOptions{RunID: c.RunID})

	if runErr != nil {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.Ui.Error(fmt.Sprintf("unable to read run: %s with: %s", c.RunID, runErr.Error()))
		return 1
	}

	// check if run can be force-cancelled at this moment
	if c.ForceCancel && !run.Actions.IsForceCancelable {
		c.addOutput("status", string(Error))
		c.addRunDetails(run)
		c.Ui.Error(fmt.Sprintf("run %s, cannot be force-cancelled", c.RunID))
		c.Ui.Output(c.closeOutput())
		return 1
	}

	// check if run can be cancelable
	if !c.ForceCancel && !run.Actions.IsCancelable {
		c.addOutput("status", string(Error))
		c.addRunDetails(run)
		c.Ui.Error(fmt.Sprintf("run %s, cannot be cancelled", c.RunID))
		c.Ui.Output(c.closeOutput())
		return 1
	}

	latestRun, cancelErr := c.cloud.CancelRun(c.Context, cloud.CancelRunOptions{
		RunID:       c.RunID,
		Comment:     c.Comment,
		ForceCancel: c.ForceCancel,
	})
	if latestRun != nil {
		run = latestRun
	}

	if cancelErr != nil {
		status := c.resolveStatus(cancelErr)
		c.addOutput("status", string(status))
		c.addRunDetails(run)
		c.Ui.Error(fmt.Sprintf("error discarding run, '%s' in Terraform Cloud: %s", c.RunID, cancelErr.Error()))
		c.Ui.Output(c.closeOutput())
		return 1
	}

	c.addOutput("status", string(Success))
	c.addRunDetails(run)
	c.Ui.Output(c.closeOutput())
	return 0
}

func (c *CancelRunCommand) addRunDetails(run *tfe.Run) {
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

func (c *CancelRunCommand) Help() string {
	helpText := `
Usage: tfci run cancel [options]
	`
	return strings.TrimSpace(helpText)
}

func (c *CancelRunCommand) Synopsis() string {
	return c.Help()
}
