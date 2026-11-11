// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/tfci/internal/cloud"
)

type PolicyShowCommand struct {
	*Meta

	RunID  string
	NoWait bool
}

func (c *PolicyShowCommand) flags() *flag.FlagSet {
	f := c.flagSet("policy show")
	f.StringVar(&c.RunID, "run", "", "HCP Terraform Run ID to check policies for.")
	f.BoolVar(&c.NoWait, "no-wait", false, "Fail immediately if policies not yet evaluated (default: wait with retry).")

	return f
}

func (c *PolicyShowCommand) Run(args []string) int {
	if err := c.setupCmd(args, c.flags()); err != nil {
		return 1
	}

	if c.RunID == "" {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.writer.ErrorResult("checking policies requires a valid run ID (use --run)")
		return 1
	}

	// Fetch policy evaluation
	eval, err := c.cloud.GetPolicyEvaluation(c.appCtx, cloud.GetPolicyEvaluationOptions{
		RunID:  c.RunID,
		NoWait: c.NoWait,
	})

	if err != nil {
		status := c.resolveStatus(err)
		c.addOutput("status", string(status))
		c.writer.ErrorResult(fmt.Sprintf("error retrieving policy evaluation for run '%s': %s", c.RunID, err.Error()))
		c.writer.OutputResult(c.closeOutput())
		return 1
	}

	c.addOutput("status", string(Success))
	c.addPolicyEvaluationDetails(eval)
	c.writer.OutputResult(c.closeOutput())
	return 0
}

func (c *PolicyShowCommand) addPolicyEvaluationDetails(eval *cloud.PolicyEvaluation) {
	if eval == nil {
		return
	}

	// Add structured outputs
	c.addOutput("run_id", eval.RunID)
	c.addOutput("total_count", fmt.Sprintf("%d", eval.TotalCount))
	c.addOutput("passed_count", fmt.Sprintf("%d", eval.PassedCount))
	c.addOutput("advisory_failed_count", fmt.Sprintf("%d", eval.AdvisoryFailedCount))
	c.addOutput("mandatory_failed_count", fmt.Sprintf("%d", eval.MandatoryFailedCount))
	c.addOutput("errored_count", fmt.Sprintf("%d", eval.ErroredCount))
	c.addOutput("requires_override", fmt.Sprintf("%t", eval.RequiresOverride))
	c.addOutput("policy_status", eval.Status)

	// Add full API response as JSON
	if eval.RawAPIResponse != nil {
		policyDetailsJSON, err := json.Marshal(eval.RawAPIResponse)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal policy details: %s", err)
		} else {
			c.addOutput("policy_details", string(policyDetailsJSON))
		}
	}

	// Add failed policies if any
	if len(eval.FailedPolicies) > 0 {
		failedPoliciesJSON, err := json.Marshal(eval.FailedPolicies)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal failed policies: %s", err)
		} else {
			c.addOutput("failed_policies", string(failedPoliciesJSON))
		}

		// Add pre-formatted list for CI/CD systems (GitLab, Jenkins, GitHub Actions, etc.)
		var policyLines []string
		for _, policy := range eval.FailedPolicies {
			policyLines = append(policyLines, fmt.Sprintf("- %s (%s)", policy.PolicyName, policy.EnforcementLevel))
		}
		c.addOutput("failed_policies_list", strings.Join(policyLines, "\n"))
	}

	// Add run link to structured output
	runLink := c.cloud.RunLinkByID(c.organization, eval.RunID)
	c.addOutput("run_link", runLink)

	// Add full payload for JSON output
	c.addOutputWithOpts("payload", eval, &outputOpts{
		stdOut:      false,
		multiLine:   true,
		platformOut: true,
	})

	// Human-readable output (when not in JSON mode)
	if !c.json {
		c.writer.Output("\nPolicy Evaluation Summary")
		c.writer.Output(fmt.Sprintf("   Total Policies: %d", eval.TotalCount))
		c.writer.Output(fmt.Sprintf("   Passed: %d", eval.PassedCount))
		c.writer.Output(fmt.Sprintf("   Failed (Advisory): %d", eval.AdvisoryFailedCount))
		c.writer.Output(fmt.Sprintf("   Failed (Mandatory): %d", eval.MandatoryFailedCount))
		c.writer.Output(fmt.Sprintf("   Errored: %d", eval.ErroredCount))

		if eval.MandatoryFailedCount > 0 {
			c.writer.Output("\nFailed Mandatory Policies:")
			for _, policy := range eval.FailedPolicies {
				if policy.EnforcementLevel == "mandatory" {
					// Display actual policy name with optional description
					policyDisplay := policy.PolicyName
					if policy.Description != "" && policy.Description != policy.PolicyName {
						policyDisplay = fmt.Sprintf("%s - %s", policy.PolicyName, policy.Description)
					}
					c.writer.Output(fmt.Sprintf("   - %s", policyDisplay))
				}
			}
		}

		if eval.AdvisoryFailedCount > 0 {
			c.writer.Output("\nFailed Advisory Policies:")
			for _, policy := range eval.FailedPolicies {
				if policy.EnforcementLevel == "advisory" {
					// Display actual policy name with optional description
					policyDisplay := policy.PolicyName
					if policy.Description != "" && policy.Description != policy.PolicyName {
						policyDisplay = fmt.Sprintf("%s - %s", policy.PolicyName, policy.Description)
					}
					c.writer.Output(fmt.Sprintf("   - %s", policyDisplay))
				}
			}
		}

		if eval.RequiresOverride {
			c.writer.Output("\nOverride Required: Policy override needed to proceed")
		} else {
			c.writer.Output("\nAll policies passed or only advisory policies failed")
		}

		// Add run link
		runLink := c.cloud.RunLinkByID(c.organization, eval.RunID)
		c.writer.Output(fmt.Sprintf("\n   View in HCP Terraform: %s", runLink))
		c.writer.Output("")
	}
}

func (c *PolicyShowCommand) Help() string {
	helpText := `
Usage: tfci [global options] policy show [options]

	Retrieves and displays Sentinel policy evaluation results for a Terraform Cloud run.
	Automatically waits for policy evaluation to complete unless --no-wait is specified.

Global Options:

	-hostname       The hostname of a Terraform Enterprise installation, if using Terraform Enterprise. Defaults to "app.terraform.io".

	-token          The token used to authenticate with HCP Terraform. Defaults to reading "TF_API_TOKEN" environment variable.

	-organization   HCP Terraform Organization Name.

Options:

	-run            HCP Terraform Run ID to check policies for (required).

	-no-wait        Fail immediately if policies not yet evaluated. Default behavior is to wait with retry until policies are evaluated.

Exit Codes:

	0   Success, policies retrieved
	1   Error (invalid run ID, API error, network failure)
	`
	return strings.TrimSpace(helpText)
}

func (c *PolicyShowCommand) Synopsis() string {
	return "Retrieves Sentinel policy evaluation results for a run"
}
