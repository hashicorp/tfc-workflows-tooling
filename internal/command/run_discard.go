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

type DiscardRunCommand struct {
	*Meta

	RunID   string
	Comment string
}

func (c *DiscardRunCommand) flags() *flag.FlagSet {
	f := c.flagSet("run discard")
	f.StringVar(&c.RunID, "run", "", "Terraform Cloud Run ID to Discard")
	f.StringVar(&c.Comment, "comment", "", "An optional comment about the run.")

	return f
}

func (c *DiscardRunCommand) Run(args []string) int {
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
		c.Ui.Error("discarding a run requires a valid run id")
		return 1
	}

	// fetch latest run details
	run, runErr := c.cloud.GetRun(c.Context, cloud.GetRunOptions{
		RunID: c.RunID,
	})
	if runErr != nil {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.Ui.Error(fmt.Sprintf("unable to read run: %s, with: %s", c.RunID, runErr.Error()))
		return 1
	}

	// first check if not able to discard run
	if !run.Actions.IsDiscardable {
		c.addOutput("status", string(Error))
		c.addRunDetails(run)
		c.Ui.Error(fmt.Sprintf("run: %s cannot be discarded", c.RunID))
		c.Ui.Output(c.closeOutput())
		return 1
	}

	latestRun, discardErr := c.cloud.DiscardRun(c.Context, cloud.DiscardRunOptions{
		RunID:   c.RunID,
		Comment: c.Comment,
	})
	// update latest run results
	if latestRun != nil {
		run = latestRun
	}

	if discardErr != nil {
		status := c.resolveStatus(discardErr)
		c.addOutput("status", string(status))
		c.addRunDetails(run)
		c.Ui.Error(fmt.Sprintf("error discarding run, '%s' in Terraform Cloud: %s", c.RunID, discardErr.Error()))
		c.Ui.Output(c.closeOutput())
		return 1
	}

	c.addOutput("status", string(Success))
	c.addRunDetails(run)
	c.Ui.Output(c.closeOutput())
	return 0
}

func (c *DiscardRunCommand) addRunDetails(run *tfe.Run) {
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

func (c *DiscardRunCommand) Help() string {
	helpText := `
Usage: tfci run discard [options]
	`
	return strings.TrimSpace(helpText)
}

func (c *DiscardRunCommand) Synopsis() string {
	return c.Help()
}
