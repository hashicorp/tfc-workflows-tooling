// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-tfe/mocks"
	"github.com/hashicorp/tfci/internal/writer"
	"github.com/mitchellh/cli"
	"go.uber.org/mock/gomock"
)

func TestWorkspaceService_ReadStateOutputs(t *testing.T) {

	testCases := []struct {
		name                   string
		orgName                string
		workspaceName          string
		ctx                    context.Context
		workspaceID            string
		tfeWorkspace           *tfe.Workspace
		tfeStateVersion        *tfe.StateVersion
		tfeStateVersionOutputs *tfe.StateVersionOutputsList
	}{
		{
			name:          "basic",
			orgName:       "abc-company",
			workspaceName: "my-workspace",
			ctx:           context.Background(),
			workspaceID:   "ws-***",
			tfeWorkspace:  &tfe.Workspace{ID: "ws-***"},
			tfeStateVersion: &tfe.StateVersion{
				ResourcesProcessed: true,
			},
			tfeStateVersionOutputs: &tfe.StateVersionOutputsList{
				Items: []*tfe.StateVersionOutput{
					{
						Name:  "image_id",
						Value: "ami-12345",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// mock workspace
			mWorkspace := mocks.NewMockWorkspaces(ctrl)
			mWorkspace.EXPECT().Read(tc.ctx, tc.orgName, tc.workspaceName).Return(
				tc.tfeWorkspace,
				nil,
			)

			// mock state version
			mockStateVersion := mocks.NewMockStateVersions(ctrl)
			mockStateVersion.EXPECT().ReadCurrent(tc.ctx, tc.workspaceID).Return(
				tc.tfeStateVersion,
				nil,
			)

			// mock state version output
			mockStateVersionOutputList := mocks.NewMockStateVersionOutputs(ctrl)
			mockStateVersionOutputList.EXPECT().ReadCurrent(tc.ctx, tc.workspaceID).Return(
				tc.tfeStateVersionOutputs,
				nil,
			)

			meta := &cloudMeta{
				tfe: &tfe.Client{
					Workspaces:          mWorkspace,
					StateVersions:       mockStateVersion,
					StateVersionOutputs: mockStateVersionOutputList,
				},
				writer: writer.NewWriter(cli.NewMockUi()),
			}
			client := NewWorkspaceService(meta)

			result, resultErr := client.ReadStateOutputs(tc.ctx, tc.orgName, tc.workspaceName)

			if resultErr != nil {
				t.Fatalf("expected %v but received %s", nil, resultErr)
			}

			if !reflect.DeepEqual(result, tc.tfeStateVersionOutputs) {
				t.Errorf("expected %v but received %v", result, tc.tfeStateVersionOutputs)
			}
		})
	}
}

func TestWorkspaceService_ReadStateOutputs_Retry(t *testing.T) {
	t.Run("test-retry-behavior", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx, orgName, workspaceName, wID := context.Background(), "test-org", "my-workspace", "ws-***"

		tfeWorkspace := &tfe.Workspace{ID: wID}
		tfeStateVersion := &tfe.StateVersion{
			ResourcesProcessed: false,
		}
		tfeStateVersionOutputs := &tfe.StateVersionOutputsList{
			Items: []*tfe.StateVersionOutput{
				{
					Name:  "image_id",
					Value: "ami-12345",
				},
			},
		}

		// mock workspace
		mWorkspace := mocks.NewMockWorkspaces(ctrl)
		mWorkspace.EXPECT().Read(ctx, orgName, workspaceName).Return(
			tfeWorkspace,
			nil,
		)

		// mock state version
		mockStateVersion := mocks.NewMockStateVersions(ctrl)
		// Assert and mock retry with resources processed set to false
		retryCall := mockStateVersion.EXPECT().ReadCurrent(ctx, wID).Return(tfeStateVersion, nil).Times(3)
		// Assert and mock retry is stopped when resources processed is set to true
		doneCall := mockStateVersion.EXPECT().ReadCurrent(ctx, wID).Return(&tfe.StateVersion{
			ResourcesProcessed: true,
		}, nil)

		// Expect retry calls before done call
		gomock.InOrder(
			retryCall,
			doneCall,
		)

		mockStateVersionOutputList := mocks.NewMockStateVersionOutputs(ctrl)
		mockStateVersionOutputList.EXPECT().ReadCurrent(ctx, wID).Return(
			tfeStateVersionOutputs,
			nil,
		)

		meta := &cloudMeta{
			tfe: &tfe.Client{
				Workspaces:          mWorkspace,
				StateVersions:       mockStateVersion,
				StateVersionOutputs: mockStateVersionOutputList,
			},
			writer: writer.NewWriter(cli.NewMockUi()),
		}
		client := NewWorkspaceService(meta)

		// invoke workspace service call
		client.ReadStateOutputs(ctx, orgName, workspaceName)
	})
}
