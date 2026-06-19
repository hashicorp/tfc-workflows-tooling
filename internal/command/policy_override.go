// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hashicorp/tfci/internal/cloud"
)

type PolicyOverrideCommand struct {
	*Meta

	RunID         string
	Justification string
}

func (c *PolicyOverrideCommand) flags() *flag.FlagSet {
	f := c.flagSet("policy override")
	f.StringVar(&c.RunID, "run", "", "HCP Terraform Run ID to override policies for.")
	f.StringVar(&c.Justification, "justification", "", "Reason for override (minimum 10 characters).")

	return f
}

func (c *PolicyOverrideCommand) Run(args []string) int {
	if err := c.setupCmd(args, c.flags()); err != nil {
		return 1
	}

	// Validate inputs
	if c.RunID == "" {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.writer.ErrorResult("overriding policies requires a valid run ID (use --run)")
		return 1
	}

	if c.Justification == "" {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.writer.ErrorResult("overriding policies requires a justification (use --justification)")
		return 1
	}

	// Apply policy override
	override, err := c.cloud.OverridePolicy(c.appCtx, cloud.OverridePolicyOptions{
		RunID:         c.RunID,
		Justification: c.Justification,
	})

	if err != nil {
		status := c.resolveStatus(err)
		c.addOutput("status", string(status))
		c.writer.ErrorResult(fmt.Sprintf("error applying policy override for run '%s': %s", c.RunID, err.Error()))
		c.writer.OutputResult(c.closeOutput())

		// Return specific exit codes for different error conditions
		if strings.Contains(err.Error(), "discarded") {
			return 2
		}
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
			return 3
		}
		return 1
	}

	c.addOutput("status", string(Success))
	c.addPolicyOverrideDetails(override)
	c.writer.OutputResult(c.closeOutput())
	return 0
}

func (c *PolicyOverrideCommand) addPolicyOverrideDetails(override *cloud.PolicyOverride) {
	if override == nil {
		return
	}

	// Add structured outputs
	c.addOutput("run_id", override.RunID)
	c.addOutput("initial_status", override.InitialStatus)
	c.addOutput("final_status", override.FinalStatus)
	c.addOutput("override_complete", fmt.Sprintf("%t", override.OverrideComplete))
	c.addOutput("justification", override.Justification)
	c.addOutput("timestamp", override.Timestamp.Format("2006-01-02T15:04:05Z07:00"))

	// Add run link to structured output
	runLink := c.cloud.RunLinkByID(c.organization, override.RunID)
	c.addOutput("run_link", runLink)

	// Add full payload for JSON output
	c.addOutputWithOpts("payload", override, &outputOpts{
		stdOut:      false,
		multiLine:   true,
		platformOut: true,
	})

	// Human-readable output (when not in JSON mode)
	if !c.json {
		c.writer.Output("\nApplying policy override...")
		c.writer.Output(fmt.Sprintf("Justification: %s", override.Justification))
		c.writer.Output("")

		c.writer.Output("Policy override applied successfully")
		c.writer.Output("Justification comment added to run")

		if override.OverrideComplete {
			c.writer.Output(fmt.Sprintf("Override complete! Run status: %s", override.FinalStatus))
		} else {
			c.writer.Output(fmt.Sprintf("Override processing... Run status: %s", override.FinalStatus))
		}

		c.writer.Output("\nNext Steps:")
		switch override.FinalStatus {
		case "policy_override", "post_plan_completed":
			c.writer.Output("- Run the Apply workflow to deploy changes")
		case "apply_queued":
			c.writer.Output("- Apply is already queued (workspace has auto-apply enabled)")
		}

		// Add run link
		runLink := c.cloud.RunLinkByID(c.organization, override.RunID)
		c.writer.Output(fmt.Sprintf("-    View run: %s", runLink))
		c.writer.Output("")
	}
}

func (c *PolicyOverrideCommand) Help() string {
	helpText := `
Usage: tfci [global options] policy override [options]

	Applies a policy override with justification to unblock deployments when
	mandatory Sentinel policies fail.

	This command should be used with caution and requires appropriate approval
	and justification. The justification is added as a comment to the run for
	audit trail purposes.

Global Options:

	-hostname       The hostname of a Terraform Enterprise installation, if using Terraform Enterprise. Defaults to "app.terraform.io".

	-token          The token used to authenticate with HCP Terraform. Defaults to reading "TF_API_TOKEN" environment variable.

	-organization   HCP Terraform Organization Name.

Options:

	-run            HCP Terraform Run ID to override (required).
	                Run must be in 'post_plan_awaiting_decision' status.

	-justification  Reason for override (required, minimum 10 characters).
	                Should reference approval source (e.g., incident ticket, change request).

Exit Codes:

	0   Override applied successfully
	1   Error (wrong status, no mandatory failures, permissions error)
	2   Run discarded during override
	3   Override timeout

Example:

	tfci policy override \
	  --run run-abc123def456 \
	  --justification "Emergency hotfix approved by CTO - Incident INC-12345"
	`
	return strings.TrimSpace(helpText)
}

func (c *PolicyOverrideCommand) Synopsis() string {
	return "Applies a policy override with justification to unblock deployments"
}
