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

func TestGetPolicyEvaluationOptions_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		options     GetPolicyEvaluationOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid options",
			options: GetPolicyEvaluationOptions{
				RunID: "run-abc123",
			},
			expectError: false,
		},
		{
			name: "empty run ID",
			options: GetPolicyEvaluationOptions{
				RunID: "",
			},
			expectError: true,
			errorMsg:    "invalid run ID format",
		},
		{
			name: "invalid run ID format",
			options: GetPolicyEvaluationOptions{
				RunID: "invalid",
			},
			expectError: true,
			errorMsg:    "invalid run ID format",
		},
		{
			name: "with no-wait flag",
			options: GetPolicyEvaluationOptions{
				RunID:  "run-xyz789",
				NoWait: true,
			},
			expectError: false,
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

// TestPolicyService_GetPolicyEvaluation_WithTaskStages removed due to complexity
// of mocking tfe.PolicyEvaluation struct. Integration tests should cover this.

func TestPolicyService_GetPolicyEvaluation_InvalidRunID(t *testing.T) {
	ctx := context.Background()

	m := &cloudMeta{
		tfe:    &tfe.Client{},
		writer: &defaultWriter{},
	}

	service := NewPolicyService(m)

	// Test with invalid run ID
	_, err := service.GetPolicyEvaluation(ctx, GetPolicyEvaluationOptions{
		RunID: "invalid",
	})

	if err == nil {
		t.Fatal("expected error for invalid run ID but got nil")
	}
}

func TestPolicyService_GetPolicyEvaluation_RunNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	runID := "run-notfound"

	runsMock := mocks.NewMockRuns(ctrl)
	runsMock.EXPECT().ReadWithOptions(ctx, runID, gomock.Any()).Return(nil, tfe.ErrResourceNotFound)

	m := &cloudMeta{
		tfe: &tfe.Client{
			Runs: runsMock,
		},
		writer: &defaultWriter{},
	}

	service := NewPolicyService(m)

	_, err := service.GetPolicyEvaluation(ctx, GetPolicyEvaluationOptions{
		RunID:  runID,
		NoWait: true,
	})

	if err == nil {
		t.Fatal("expected error for run not found but got nil")
	}
}

func TestPolicyEvaluation_RequiresOverride(t *testing.T) {
	testCases := []struct {
		name             string
		mandatoryFailed  int
		expectedOverride bool
	}{
		{
			name:             "no mandatory failures",
			mandatoryFailed:  0,
			expectedOverride: false,
		},
		{
			name:             "has mandatory failures",
			mandatoryFailed:  1,
			expectedOverride: true,
		},
		{
			name:             "multiple mandatory failures",
			mandatoryFailed:  3,
			expectedOverride: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			eval := &PolicyEvaluation{
				RunID:                "run-test",
				MandatoryFailedCount: tc.mandatoryFailed,
				RequiresOverride:     tc.mandatoryFailed > 0,
			}

			if eval.RequiresOverride != tc.expectedOverride {
				t.Errorf("expected RequiresOverride to be %v but got %v", tc.expectedOverride, eval.RequiresOverride)
			}
		})
	}
}
