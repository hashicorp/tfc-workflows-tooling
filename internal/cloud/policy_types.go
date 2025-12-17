// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"fmt"
	"regexp"
	"time"
)

// Pre-compiled regex for TFC resource ID validation
var validIDPattern = regexp.MustCompile(`^[a-z]+-[a-zA-Z0-9]+$`)

// Enforcement levels for policies
const (
	EnforcementMandatory = "mandatory"
	EnforcementAdvisory  = "advisory"
)

// Policy statuses
const (
	PolicyStatusFailed  = "failed"
	PolicyStatusErrored = "errored"
	PolicyStatusPassed  = "passed"
)

// Run statuses for policy operations
const (
	RunStatusPostPlanAwaitingDecision = "post_plan_awaiting_decision"
	RunStatusPolicyOverride           = "policy_override"
	RunStatusPostPlanCompleted        = "post_plan_completed"
	RunStatusApplyQueued              = "apply_queued"
	RunStatusDiscarded                = "discarded"
	RunStatusErrored                  = "errored"
)

// MinJustificationLength is the minimum required length for override justification
const MinJustificationLength = 10

// PolicyEvaluation represents normalized policy evaluation results
type PolicyEvaluation struct {
	RunID                string         `json:"run_id"`
	PolicyStageID        string         `json:"policy_stage_id,omitempty"`
	PolicyCheckID        string         `json:"policy_check_id,omitempty"`
	TotalCount           int            `json:"total_count"`
	PassedCount          int            `json:"passed_count"`
	AdvisoryFailedCount  int            `json:"advisory_failed_count"`
	MandatoryFailedCount int            `json:"mandatory_failed_count"`
	ErroredCount         int            `json:"errored_count"`
	FailedPolicies       []PolicyDetail `json:"failed_policies"`
	Status               string         `json:"status"`
	RequiresOverride     bool           `json:"requires_override"`
	// RawAPIResponse contains the full API response from TFC (TaskStage or PolicyCheck)
	RawAPIResponse any `json:"raw_api_response,omitempty"`
}

// Validate checks PolicyEvaluation data integrity
func (pe *PolicyEvaluation) Validate() error {
	if !validStringID(pe.RunID) {
		return ErrInvalidRunID
	}

	if pe.PolicyStageID == "" && pe.PolicyCheckID == "" {
		return ErrPolicyIDRequired
	}

	if pe.PolicyStageID != "" && pe.PolicyCheckID != "" {
		return ErrPolicyIDMutualExclusive
	}

	if pe.PolicyStageID != "" && !validStringID(pe.PolicyStageID) {
		return ErrInvalidPolicyStageID
	}

	if pe.PolicyCheckID != "" && !validStringID(pe.PolicyCheckID) {
		return ErrInvalidPolicyCheckID
	}

	if pe.TotalCount < 0 || pe.PassedCount < 0 || pe.AdvisoryFailedCount < 0 ||
		pe.MandatoryFailedCount < 0 || pe.ErroredCount < 0 {
		return ErrNegativeCount
	}

	expectedTotal := pe.PassedCount + pe.AdvisoryFailedCount + pe.MandatoryFailedCount + pe.ErroredCount
	if pe.TotalCount != expectedTotal {
		return ErrCountMismatch
	}

	if pe.RequiresOverride != (pe.MandatoryFailedCount > 0) {
		return ErrOverrideMismatch
	}

	return nil
}

// PolicyDetail represents individual policy failure information
type PolicyDetail struct {
	PolicyName       string `json:"policy_name"`
	EnforcementLevel string `json:"enforcement_level"`
	Status           string `json:"status"`
	Description      string `json:"description,omitempty"`
}

// Validate checks PolicyDetail data integrity
func (pd *PolicyDetail) Validate() error {
	if pd.PolicyName == "" {
		return ErrEmptyPolicyName
	}

	if pd.EnforcementLevel != EnforcementMandatory && pd.EnforcementLevel != EnforcementAdvisory {
		return ErrInvalidEnforcementLevel
	}

	if pd.Status != PolicyStatusFailed && pd.Status != PolicyStatusErrored {
		return ErrInvalidPolicyStatus
	}

	return nil
}

// PolicyOverride represents a policy override action
type PolicyOverride struct {
	RunID            string    `json:"run_id"`
	PolicyStageID    string    `json:"policy_stage_id,omitempty"`
	PolicyCheckID    string    `json:"policy_check_id,omitempty"`
	Justification    string    `json:"justification"`
	InitialStatus    string    `json:"initial_status"`
	FinalStatus      string    `json:"final_status"`
	OverrideComplete bool      `json:"override_complete"`
	Timestamp        time.Time `json:"timestamp"`
}

// Validate checks PolicyOverride data integrity
func (po *PolicyOverride) Validate() error {
	if !validStringID(po.RunID) {
		return ErrInvalidRunID
	}

	if po.PolicyStageID == "" && po.PolicyCheckID == "" {
		return ErrPolicyIDRequired
	}

	if po.PolicyStageID != "" && po.PolicyCheckID != "" {
		return ErrPolicyIDMutualExclusive
	}

	if po.PolicyStageID != "" && !validStringID(po.PolicyStageID) {
		return ErrInvalidPolicyStageID
	}

	if po.PolicyCheckID != "" && !validStringID(po.PolicyCheckID) {
		return ErrInvalidPolicyCheckID
	}

	if po.Justification == "" {
		return ErrInvalidJustification
	}

	if po.InitialStatus != RunStatusPostPlanAwaitingDecision {
		return ErrInvalidInitialStatus
	}

	validFinalStatuses := map[string]bool{
		RunStatusPolicyOverride:    true,
		RunStatusPostPlanCompleted: true,
		RunStatusApplyQueued:       true,
		RunStatusDiscarded:         true,
		RunStatusErrored:           true,
	}
	if !validFinalStatuses[po.FinalStatus] {
		return ErrInvalidFinalStatus
	}

	return nil
}

// GetPolicyEvaluationOptions configures policy evaluation retrieval
type GetPolicyEvaluationOptions struct {
	RunID  string // Required: TFC run ID
	NoWait bool   // Optional: Fail fast if policies not yet evaluated
}

// Validate checks if options are valid
func (o GetPolicyEvaluationOptions) Validate() error {
	if !validStringID(o.RunID) {
		return ErrInvalidRunID
	}
	return nil
}

// OverridePolicyOptions configures policy override operation
type OverridePolicyOptions struct {
	RunID         string // Required: TFC run ID
	Justification string // Required: Override reason
}

// Validate checks if options are valid
func (o OverridePolicyOptions) Validate() error {
	if !validStringID(o.RunID) {
		return ErrInvalidRunID
	}
	if o.Justification == "" {
		return ErrInvalidJustification
	}
	if len(o.Justification) < MinJustificationLength {
		return fmt.Errorf("%w: must be at least %d characters", ErrInvalidJustification, MinJustificationLength)
	}
	return nil
}

// validStringID checks if a string is a valid TFC resource ID
func validStringID(id string) bool {
	if id == "" {
		return false
	}
	return validIDPattern.MatchString(id)
}
