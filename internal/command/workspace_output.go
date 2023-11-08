// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"flag"
	"fmt"
	"strings"
)

type WorkspaceOutputCommand struct {
	*Meta

	Workspace string
}

type WorkspaceOutput struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

func (c *WorkspaceOutputCommand) flags() *flag.FlagSet {
	f := c.flagSet("state output")
	f.StringVar(&c.Workspace, "workspace", "", "The name of the Terraform Cloud Workspace.")

	return f
}

func (c *WorkspaceOutputCommand) Run(args []string) int {
	if err := c.setupCmd(args, c.flags()); err != nil {
		return 1
	}

	// validate workspace name was supplied as argument
	if c.Workspace == "" {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.writer.ErrorResult("error workspace output list requires a workspace name")
		return 1
	}

	svoList, svoErr := c.cloud.ReadStateOutputs(c.appCtx, c.organization, c.Workspace)
	if svoErr != nil {
		status := c.resolveStatus(svoErr)
		c.addOutput("status", string(status))
		c.closeOutput()
		c.writer.ErrorResult(fmt.Sprintf("error retrieving workspace state version outputs: %s\n", svoErr.Error()))
		return 1
	}

	workspaceOutputs := []*WorkspaceOutput{}
	for _, svo := range svoList.Items {
		workspaceOutputs = append(workspaceOutputs, &WorkspaceOutput{
			Name:  svo.Name,
			Value: svo.Value,
		})
	}

	c.addOutputWithOpts("outputs", workspaceOutputs, &outputOpts{
		stdOut:      true,
		multiLine:   true,
		platformOut: true,
	})
	c.addOutput("status", string(Success))
	c.writer.OutputResult(c.closeOutput())
	return 0
}

func (c *WorkspaceOutputCommand) Help() string {
	helpText := `
Usage: tfci [global options] workspace outputs [options]

	Returns current state version outputs for a workspace.

Global Options:

	-hostname       The hostname of a Terraform Enterprise installation, if using Terraform Enterprise. Defaults to "app.terraform.io".

	-token          The token used to authenticate with Terraform Cloud. Defaults to reading "TF_API_TOKEN" environment variable.

	-organization   Terraform Cloud Organization Name.

Options:

	-workspace            Existing Terraform Cloud Workspace.
	`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceOutputCommand) Synopsis() string {
	return "Returns current state version outputs for a workspace."
}
