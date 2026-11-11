// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"
	"testing"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-tfe/mocks"
	"go.uber.org/mock/gomock"
)

func TestOverridePolicyOptions_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		options     OverridePolicyOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid options",
			options: OverridePolicyOptions{
				RunID:         "run-abc123",
				Justification: "Emergency deployment approved",
			},
			expectError: false,
		},
		{
			name: "empty run ID",
			options: OverridePolicyOptions{
				RunID:         "",
				Justification: "Some justification",
			},
			expectError: true,
			errorMsg:    "invalid run ID format",
		},
		{
			name: "invalid run ID format",
			options: OverridePolicyOptions{
				RunID:         "invalid",
				Justification: "Some justification",
			},
			expectError: true,
			errorMsg:    "invalid run ID format",
		},
		{
			name: "empty justification",
			options: OverridePolicyOptions{
				RunID:         "run-abc123",
				Justification: "",
			},
			expectError: true,
			errorMsg:    "invalid justification",
		},
		{
			name: "short justification rejected",
			options: OverridePolicyOptions{
				RunID:         "run-abc123",
				Justification: "ok",
			},
			expectError: true,
			errorMsg:    "invalid justification: must be at least 10 characters",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.options.Validate()

			if tc.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}

			if tc.expectError && err != nil && tc.errorMsg != "" {
				if err.Error() != tc.errorMsg {
					t.Errorf("expected error message '%s' but got '%s'", tc.errorMsg, err.Error())
				}
			}
		})
	}
}

func TestPolicyOverride_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		result      PolicyOverride
		expectError bool
	}{
		{
			name: "valid with policy stage",
			result: PolicyOverride{
				RunID:          "run-abc123",
				PolicyStageID:  "ts-123",
				PolicyCheckID:  "",
				Justification:  "Emergency fix",
				InitialStatus:  "post_plan_awaiting_decision",
				FinalStatus:    "policy_override",
			},
			expectError: false,
		},
		{
			name: "valid with policy check",
			result: PolicyOverride{
				RunID:          "run-abc123",
				PolicyStageID:  "",
				PolicyCheckID:  "polchk-123",
				Justification:  "Approved override",
				InitialStatus:  "post_plan_awaiting_decision",
				FinalStatus:    "post_plan_completed",
			},
			expectError: false,
		},
		{
			name: "missing both stage and check ID",
			result: PolicyOverride{
				RunID:          "run-abc123",
				PolicyStageID:  "",
				PolicyCheckID:  "",
				Justification:  "Test",
				InitialStatus:  "post_plan_awaiting_decision",
				FinalStatus:    "policy_override",
			},
			expectError: true,
		},
		{
			name: "empty justification",
			result: PolicyOverride{
				RunID:          "run-abc123",
				PolicyStageID:  "ts-123",
				PolicyCheckID:  "",
				Justification:  "",
				InitialStatus:  "post_plan_awaiting_decision",
				FinalStatus:    "policy_override",
			},
			expectError: true,
		},
		{
			name: "invalid initial status",
			result: PolicyOverride{
				RunID:          "run-abc123",
				PolicyStageID:  "ts-123",
				PolicyCheckID:  "",
				Justification:  "Test",
				InitialStatus:  "planning",
				FinalStatus:    "policy_override",
			},
			expectError: true,
		},
		{
			name: "invalid final status",
			result: PolicyOverride{
				RunID:          "run-abc123",
				PolicyStageID:  "ts-123",
				PolicyCheckID:  "",
				Justification:  "Test",
				InitialStatus:  "post_plan_awaiting_decision",
				FinalStatus:    "invalid_status",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.result.Validate()

			if tc.expectError && err == nil {
				t.Errorf("expected error but got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestPolicyService_ValidateOverrideEligibility(t *testing.T) {
	testCases := []struct {
		name        string
		runStatus   tfe.RunStatus
		expectError bool
	}{
		{
			name:        "valid status post_plan_awaiting_decision",
			runStatus:   "post_plan_awaiting_decision",
			expectError: false,
		},
		{
			name:        "invalid status planned",
			runStatus:   "planned",
			expectError: true,
		},
		{
			name:        "invalid status applied",
			runStatus:   "applied",
			expectError: true,
		},
		{
			name:        "invalid status policy_checked",
			runStatus:   "policy_checked",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			runID := "run-test123"

			runsMock := mocks.NewMockRuns(ctrl)
			runsMock.EXPECT().Read(ctx, runID).Return(&tfe.Run{
				ID:     runID,
				Status: tc.runStatus,
			}, nil)

			m := &cloudMeta{
				tfe: &tfe.Client{
					Runs: runsMock,
				},
				writer: &defaultWriter{},
			}

			service := &policyService{cloudMeta: m}

			_, err := service.validateOverrideEligibility(ctx, runID)

			if tc.expectError && err == nil {
				t.Errorf("expected error for status %s but got nil", tc.runStatus)
			}

			if !tc.expectError && err != nil {
				t.Errorf("expected no error for status %s but got: %v", tc.runStatus, err)
			}
		})
	}
}

func TestPolicyService_OverridePolicy_RunNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	runID := "run-notfound"

	runsMock := mocks.NewMockRuns(ctrl)
	runsMock.EXPECT().Read(ctx, runID).Return(nil, tfe.ErrResourceNotFound)

	m := &cloudMeta{
		tfe: &tfe.Client{
			Runs: runsMock,
		},
		writer: &defaultWriter{},
	}

	service := NewPolicyService(m)

	_, err := service.OverridePolicy(ctx, OverridePolicyOptions{
		RunID:         runID,
		Justification: "Test justification for override",
	})

	if err == nil {
		t.Fatal("expected error for run not found but got nil")
	}
}

func TestPolicyService_OverridePolicy_InvalidOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	m := &cloudMeta{
		tfe:    &tfe.Client{},
		writer: &defaultWriter{},
	}

	service := NewPolicyService(m)

	testCases := []struct {
		name    string
		options OverridePolicyOptions
	}{
		{
			name: "empty run ID",
			options: OverridePolicyOptions{
				RunID:         "",
				Justification: "Test",
			},
		},
		{
			name: "empty justification",
			options: OverridePolicyOptions{
				RunID:         "run-abc123",
				Justification: "",
			},
		},
		{
			name: "invalid run ID format",
			options: OverridePolicyOptions{
				RunID:         "invalid",
				Justification: "Test",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := service.OverridePolicy(ctx, tc.options)

			if err == nil {
				t.Errorf("expected error for %s but got nil", tc.name)
			}
		})
	}
}
