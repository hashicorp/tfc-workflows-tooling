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

	PlanID   string
	Location string
}

func (c *OutputPlanCommand) flags() *flag.FlagSet {
	f := c.flagSet("plan output")
	f.StringVar(&c.PlanID, "plan", "", "The plan ID to retrieve JSON execution plan.")
	f.StringVar(&c.Location, "location", "", "Location of where to persist the plan execution file.")

	return f
}

func (c *OutputPlanCommand) Run(args []string) int {
	flags := c.flags()
	if err := flags.Parse(args); err != nil {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.Ui.Error(fmt.Sprintf("error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	plan, pErr := c.cloud.GetPlan(c.Context, c.PlanID)
	if pErr != nil {
		c.addOutput("status", string(Error))
		c.addPlanDetails(plan)
		c.Ui.Error(fmt.Sprintf("error retrieving plan data %s\n", pErr.Error()))
		c.Ui.Output(c.closeOutput())
		return 1
	}

	c.addOutput("status", string(Success))
	c.addPlanDetails(plan)
	c.Ui.Output(c.closeOutput())
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

	planJson, _ := outputJson(plan)
	c.addOutput("payload", planJson)
}

func (c *OutputPlanCommand) Help() string {
	helpText := `
Usage: tfci plan output [options]
	`
	return strings.TrimSpace(helpText)
}

func (c *OutputPlanCommand) Synopsis() string {
	return c.Help()
}
