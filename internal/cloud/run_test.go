// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-tfe/mocks"
)

func TestRunService_CreateRun(t *testing.T) {
	testCases := []struct {
		name             string
		orgName          string
		workspaceName    string
		ctx              context.Context
		tfeWorkspace     *tfe.Workspace
		tfeConfigVersion *tfe.ConfigurationVersion
		tfeRun           *tfe.Run
		statusChanges    []tfe.RunStatus
		finalStatus      tfe.RunStatus
	}{
		{
			name:          "plan-only-run",
			orgName:       "test",
			workspaceName: "my-workspace",
			ctx:           context.Background(),
			tfeWorkspace:  &tfe.Workspace{ID: "ws-***"},
			tfeConfigVersion: &tfe.ConfigurationVersion{
				ID:     "cv-***",
				Status: tfe.ConfigurationUploaded,
			},
			tfeRun: &tfe.Run{
				ID:       "run-***",
				PlanOnly: true,
			},
			statusChanges: []tfe.RunStatus{
				tfe.RunPlanning,
				tfe.RunPlanning,
				tfe.RunCostEstimated,
				tfe.RunPolicyChecked,
			},
			finalStatus: tfe.RunPlannedAndFinished,
		},
		{
			name:          "auto-apply-run",
			orgName:       "test",
			workspaceName: "my-workspace",
			ctx:           context.Background(),
			tfeWorkspace:  &tfe.Workspace{ID: "ws-***"},
			tfeConfigVersion: &tfe.ConfigurationVersion{
				ID:     "cv-***",
				Status: tfe.ConfigurationUploaded,
			},
			tfeRun: &tfe.Run{
				ID:        "run-***",
				AutoApply: true,
				CostEstimate: &tfe.CostEstimate{
					ID: "cost-******",
				},
				PolicyChecks: []*tfe.PolicyCheck{
					{ID: "pol-****"},
				},
			},
			statusChanges: []tfe.RunStatus{
				tfe.RunPlanning,
				tfe.RunPlanning,
				tfe.RunCostEstimated,
				tfe.RunPolicyChecked,
			},
			finalStatus: tfe.RunApplied,
		},
		{
			name:          "confirmable-run",
			orgName:       "test",
			workspaceName: "my-workspace",
			ctx:           context.Background(),
			tfeWorkspace:  &tfe.Workspace{ID: "ws-***"},
			tfeConfigVersion: &tfe.ConfigurationVersion{
				ID:     "cv-***",
				Status: tfe.ConfigurationUploaded,
			},
			tfeRun: &tfe.Run{
				ID: "run-***",
				CostEstimate: &tfe.CostEstimate{
					ID: "cost-******",
				},
				PolicyChecks: []*tfe.PolicyCheck{
					{ID: "pol-****"},
				},
			},
			statusChanges: []tfe.RunStatus{
				tfe.RunPlanning,
				tfe.RunPlanned,
				tfe.RunCostEstimated,
			},
			finalStatus: tfe.RunPolicyChecked,
		},
	}

	for _, tc := range testCases {
		// reassign loop variable to prevent scope capture
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// allow these test cases to run in parallel
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// mock workspace service
			workspaceMock := mocks.NewMockWorkspaces(ctrl)
			workspaceMock.EXPECT().Read(tc.ctx, tc.orgName, tc.workspaceName).Return(
				tc.tfeWorkspace,
				nil,
			)
			// mock configuration service
			configVersionMock := mocks.NewMockConfigurationVersions(ctrl)
			configVersionMock.EXPECT().Read(tc.ctx, tc.tfeConfigVersion.ID).Return(
				tc.tfeConfigVersion,
				nil,
			)

			// mock runs service
			runMessage := ""
			runsMock := mocks.NewMockRuns(ctrl)
			runsMock.EXPECT().Create(tc.ctx, tfe.RunCreateOptions{
				ConfigurationVersion: tc.tfeConfigVersion,
				Workspace:            tc.tfeWorkspace,
				PlanOnly:             tfe.Bool(tc.tfeRun.PlanOnly),
				Message:              &runMessage,
				Variables:            []*tfe.RunVariable{},
			}).Return(tc.tfeRun, nil)

			//
			goMockCalls := []*gomock.Call{}
			for _, status := range tc.statusChanges {
				call := runsMock.EXPECT().ReadWithOptions(
					tc.ctx,
					tc.tfeRun.ID,
					&tfe.RunReadOptions{Include: []tfe.RunIncludeOpt{"cost_estimate", "plan"}}).Return(&tfe.Run{
					ID:     tc.tfeRun.ID,
					Status: status,
				}, nil)
				goMockCalls = append(goMockCalls, call)
			}

			doneCall := runsMock.EXPECT().ReadWithOptions(tc.ctx, tc.tfeRun.ID, &tfe.RunReadOptions{
				Include: []tfe.RunIncludeOpt{"cost_estimate", "plan"},
			}).Return(&tfe.Run{
				ID:     tc.tfeRun.ID,
				Status: tc.finalStatus,
			}, nil)
			goMockCalls = append(goMockCalls, doneCall)

			// Verify retry order behavior
			gomock.InOrder(
				goMockCalls...,
			)

			client := &runService{
				tfe: &tfe.Client{
					Workspaces:            workspaceMock,
					ConfigurationVersions: configVersionMock,
					Runs:                  runsMock,
				},
			}

			_, err := client.CreateRun(tc.ctx, CreateRunOptions{
				Organization:           tc.orgName,
				ConfigurationVersionID: tc.tfeConfigVersion.ID,
				Workspace:              tc.workspaceName,
				Message:                "",
				PlanOnly:               tc.tfeRun.PlanOnly,
				RunVariables:           []*tfe.RunVariable{},
			})

			if err != nil {
				t.Fatalf("expected %v but received %s", nil, err)
			}
		})
	}
}
