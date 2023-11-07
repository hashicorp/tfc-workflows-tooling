// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-tfe/mocks"
)

func TestUpload(t *testing.T) {
	type fields struct {
		Client *tfe.Client
		writer Writer
	}

	type args struct {
		ctx     context.Context
		options UploadOptions
	}

	cv := &tfe.ConfigurationVersion{
		ID:        "cv-1",
		UploadURL: "cv.com",
		Status:    tfe.ConfigurationUploaded,
	}

	writer := &defaultWriter{}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wsRead      bool
		ws          *tfe.Workspace
		wsErr       error
		cvCreate    bool
		cvUpload    bool
		cvRead      bool
		cv          *tfe.ConfigurationVersion
		cvErr       error
		cvCreateErr error
		cvUploadErr error
		want        *tfe.ConfigurationVersion
		wantErr     bool
	}{
		{
			name: "upload success",
			fields: fields{
				Client: &tfe.Client{},
				writer: writer,
			},
			args: args{
				ctx: context.Background(),
				options: UploadOptions{
					Organization:           "my-org",
					Workspace:              "my-ws",
					ConfigurationDirectory: "dir/",
					Speculative:            false,
				},
			},
			wsRead: true,
			ws: &tfe.Workspace{
				ID: "my-ws",
			},
			wsErr:       nil,
			cvCreate:    true,
			cvUpload:    true,
			cvRead:      true,
			cv:          cv,
			cvCreateErr: nil,
			cvUploadErr: nil,
			want:        cv,
			wantErr:     false,
		},
		{
			name: "workspace read fails",
			fields: fields{
				Client: &tfe.Client{},
				writer: writer,
			},
			args: args{
				ctx: context.Background(),
				options: UploadOptions{
					Organization:           "my-org",
					Workspace:              "my-ws",
					ConfigurationDirectory: "dir/",
					Speculative:            false,
				},
			},
			wsRead:      true,
			ws:          nil,
			wsErr:       errors.New(`workspace error`),
			cv:          nil,
			cvCreateErr: nil,
			cvUploadErr: nil,
			want:        nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockWs := mocks.NewMockWorkspaces(ctrl)
			if tt.wsRead {
				mockWs.EXPECT().Read(
					tt.args.ctx,
					tt.args.options.Organization,
					tt.args.options.Workspace,
				).Return(tt.ws, tt.wsErr)
			}

			mockCv := mocks.NewMockConfigurationVersions(ctrl)
			if tt.cvCreate {
				mockCv.EXPECT().Create(tt.args.ctx, tt.args.options.Workspace, gomock.Any()).Return(tt.cv, tt.cvCreateErr)
			}

			if tt.cvUpload {
				mockCv.EXPECT().Upload(tt.args.ctx, tt.cv.UploadURL, tt.args.options.ConfigurationDirectory).Return(tt.cvUploadErr)

			}
			if tt.cvRead {
				mockCv.EXPECT().Read(tt.args.ctx, tt.cv.ID).Return(tt.cv, tt.cvCreateErr)
			}

			m := &cloudMeta{
				tfe:    tt.fields.Client,
				writer: writer,
			}
			m.tfe.Workspaces = mockWs
			m.tfe.ConfigurationVersions = mockCv
			client := NewConfigVersionService(m)

			got, err := client.UploadConfig(tt.args.ctx, tt.args.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("Upload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Upload() got = %v, want %v", got, tt.want)
			}
		})
	}
}
