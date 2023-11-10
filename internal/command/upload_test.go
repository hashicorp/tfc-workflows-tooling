// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"testing"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/tfci/internal/cloud"
	"github.com/hashicorp/tfci/internal/environment"
	"github.com/hashicorp/tfci/internal/writer"
	"github.com/mitchellh/cli"
)

type SuccessfulUploader struct {
	configurationVersion *tfe.ConfigurationVersion
}

func (s *SuccessfulUploader) UploadConfig(_ context.Context, _ cloud.UploadOptions) (*tfe.ConfigurationVersion, error) {
	return s.configurationVersion, nil
}

func meta(cv *tfe.ConfigurationVersion) *Meta {
	ctx := context.Background()
	ui := cli.NewMockUi()
	writer := writer.NewWriter(ui)
	cloudService := cloud.NewCloud(&tfe.Client{}, writer)
	cloudService.ConfigVersionService = &SuccessfulUploader{
		configurationVersion: cv,
	}
	env := &environment.CI{}
	meta := NewMetaOpts(ctx, cloudService, env, WithWriter(writer))
	return meta
}

func TestUploadConfigurationCommandRun(t *testing.T) {
	type fields struct {
		Meta        *Meta
		Workspace   string
		Directory   string
		Speculative bool
		Provisional bool
	}

	type args struct {
		args []string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "success-path",
			fields: fields{
				Meta: meta(&tfe.ConfigurationVersion{
					ID: "cv-1",
				}),
				Workspace:   "ws-1",
				Directory:   "dir/",
				Speculative: false,
				Provisional: false,
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &UploadConfigurationCommand{
				Meta:        tt.fields.Meta,
				Workspace:   tt.fields.Workspace,
				Directory:   tt.fields.Directory,
				Speculative: tt.fields.Speculative,
				Provisional: tt.fields.Provisional,
			}
			if got := c.Run(tt.args.args); got != tt.want {
				t.Errorf("Run() = %v, want %v", got, tt.want)
			}
		})
	}
}
