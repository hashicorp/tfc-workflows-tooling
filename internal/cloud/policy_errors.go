// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import "errors"

var (
	// ErrInvalidRunID indicates run ID format is invalid
	ErrInvalidRunID = errors.New("invalid run ID format")

	// ErrInvalidJustification indicates justification is missing or too short
	ErrInvalidJustification = errors.New("invalid justification")

	// ErrInvalidRunStatus indicates run is not in correct status for operation
	ErrInvalidRunStatus = errors.New("run status does not allow this operation")

	// ErrNoPolicyCheck indicates run has no policy check or task stage
	ErrNoPolicyCheck = errors.New("run has no policy evaluation")

	// ErrPolicyPending indicates policies are still being evaluated (only with NoWait=true)
	ErrPolicyPending = errors.New("policy evaluation still in progress")

	// ErrRunNotFound indicates run does not exist
	ErrRunNotFound = errors.New("run not found")

	// ErrPermissionDenied indicates insufficient permissions
	ErrPermissionDenied = errors.New("insufficient permissions for this operation")

	// ErrPolicyIDRequired indicates neither PolicyStageID nor PolicyCheckID is set
	ErrPolicyIDRequired = errors.New("either PolicyStageID or PolicyCheckID must be set")

	// ErrPolicyIDMutualExclusive indicates both PolicyStageID and PolicyCheckID are set
	ErrPolicyIDMutualExclusive = errors.New("PolicyStageID and PolicyCheckID are mutually exclusive")

	// ErrInvalidPolicyStageID indicates PolicyStageID format is invalid
	ErrInvalidPolicyStageID = errors.New("invalid policy stage ID format")

	// ErrInvalidPolicyCheckID indicates PolicyCheckID format is invalid
	ErrInvalidPolicyCheckID = errors.New("invalid policy check ID format")

	// ErrNegativeCount indicates a count field has a negative value
	ErrNegativeCount = errors.New("counts must be non-negative")

	// ErrCountMismatch indicates total count does not match sum of individual counts
	ErrCountMismatch = errors.New("total count does not match sum of individual counts")

	// ErrOverrideMismatch indicates RequiresOverride does not match MandatoryFailedCount
	ErrOverrideMismatch = errors.New("RequiresOverride mismatch with MandatoryFailedCount")

	// ErrEmptyPolicyName indicates policy name is empty
	ErrEmptyPolicyName = errors.New("policy name must not be empty")

	// ErrInvalidEnforcementLevel indicates invalid enforcement level value
	ErrInvalidEnforcementLevel = errors.New("invalid enforcement level")

	// ErrInvalidPolicyStatus indicates invalid policy status value
	ErrInvalidPolicyStatus = errors.New("invalid policy status")

	// ErrInvalidInitialStatus indicates initial status is not valid for override
	ErrInvalidInitialStatus = errors.New("invalid initial status for policy override")

	// ErrInvalidFinalStatus indicates final status is not a valid post-override status
	ErrInvalidFinalStatus = errors.New("invalid final status for policy override")
)
