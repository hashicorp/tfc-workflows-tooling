// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/go-tfe"
)

type OutputPlanCommand struct {
	*Meta

	PlanID string
}

func (c *OutputPlanCommand) flags() *flag.FlagSet {
	f := c.flagSet("plan output")
	f.StringVar(&c.PlanID, "plan", "", "The plan ID to retrieve JSON execution plan.")

	return f
}

func (c *OutputPlanCommand) Run(args []string) int {
	flags := c.flags()
	if err := flags.Parse(args); err != nil {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.ui.Error(fmt.Sprintf("error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	plan, pErr := c.cloud.GetPlan(c.appCtx, c.PlanID)
	if pErr != nil {
		c.addOutput("status", string(Error))
		c.addPlanDetails(plan)
		c.ui.Error(fmt.Sprintf("error retrieving plan data %s\n", pErr.Error()))
		c.ui.Output(c.closeOutput())
		return 1
	}

	c.addOutput("status", string(Success))
	c.addPlanDetails(plan)
	c.ui.Output(c.closeOutput())
	return 0
}

func (c *OutputPlanCommand) addPlanDetails(plan *tfe.Plan) {
	if plan == nil {
		return
	}
	c.addOutput("plan_id", plan.ID)
	c.addOutput("plan_status", string(plan.Status))
	c.addOutput("add", fmt.Sprint(plan.ResourceAdditions))
	c.addOutput("change", fmt.Sprint(plan.ResourceChanges))
	c.addOutput("destroy", fmt.Sprint(plan.ResourceDestructions))

	c.addOutputWithOpts("payload", plan, &outputOpts{
		stdOut:      false,
		multiLine:   true,
		platformOut: true,
	})
}

func (c *OutputPlanCommand) Help() string {
	helpText := `
Usage: tfci [global options] plan output [options]

	Returns the plan details for the provided Plan ID.

Global Options:

	-hostname       The hostname of a Terraform Enterprise installation, if using Terraform Enterprise. Defaults to "app.terraform.io".

	-token          The token used to authenticate with Terraform Cloud. Defaults to reading "TF_API_TOKEN" environment variable.

	-organization   Terraform Cloud Organization Name.

Options:

	-plan           Returns the plan details for the provided Plan ID.
	`
	return strings.TrimSpace(helpText)
}

func (c *OutputPlanCommand) Synopsis() string {
	return "Returns the plan details for the provided Plan ID"
}
