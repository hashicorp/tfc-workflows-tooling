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
	if err := c.setupCmd(args, c.flags()); err != nil {
		return 1
	}

	if c.RunID == "" {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.writer.ErrorResult("discarding a run requires a valid run id")
		return 1
	}

	// fetch latest run details
	run, runErr := c.cloud.GetRun(c.appCtx, cloud.GetRunOptions{
		RunID: c.RunID,
	})
	if runErr != nil {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.writer.ErrorResult(fmt.Sprintf("unable to read run: %s, with: %s", c.RunID, runErr.Error()))
		return 1
	}

	// first check if not able to discard run
	if !run.Actions.IsDiscardable {
		c.addOutput("status", string(Error))
		c.addRunDetails(run)
		c.writer.ErrorResult(fmt.Sprintf("run: %s cannot be discarded", c.RunID))
		c.writer.OutputResult(c.closeOutput())
		return 1
	}

	latestRun, discardErr := c.cloud.DiscardRun(c.appCtx, cloud.DiscardRunOptions{
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
		c.writer.ErrorResult(fmt.Sprintf("error discarding run, '%s' in Terraform Cloud: %s", c.RunID, discardErr.Error()))
		c.writer.OutputResult(c.closeOutput())
		return 1
	}

	c.addOutput("status", string(Success))
	c.addRunDetails(run)
	c.writer.OutputResult(c.closeOutput())
	return 0
}

func (c *DiscardRunCommand) addRunDetails(run *tfe.Run) {
	if run == nil {
		return
	}
	link, _ := c.cloud.RunLink(c.appCtx, c.organization, run)
	if link != "" {
		c.addOutput("run_link", link)
	}
	c.addOutput("run_id", run.ID)
	c.addOutput("run_status", string(run.Status))
}

func (c *DiscardRunCommand) Help() string {
	helpText := `
Usage: tfci [global options] run discard [options]

	Skips any remaining work on runs that are paused waiting for confirmation or priority.

Global Options:

	-hostname       The hostname of a Terraform Enterprise installation, if using Terraform Enterprise. Defaults to "app.terraform.io".

	-token          The token used to authenticate with Terraform Cloud. Defaults to reading "TF_API_TOKEN" environment variable.

	-organization   Terraform Cloud Organization Name.

Options:

	-run         Existing Terraform Cloud Run ID to Discard.

	-comment     An optional comment about the run.
	`
	return strings.TrimSpace(helpText)
}

func (c *DiscardRunCommand) Synopsis() string {
	return "Skips any remaining work on runs that are paused waiting for confirmation or priority"
}
