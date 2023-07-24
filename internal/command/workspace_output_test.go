// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/tfci/internal/cloud"
	"github.com/hashicorp/tfci/internal/environment"
	"github.com/mitchellh/cli"
)

type WorkspaceOutputReader struct {
	svo *tfe.StateVersionOutputsList
}

func (w *WorkspaceOutputReader) ReadStateOutputs(_ context.Context, orgName string, wName string) (*tfe.StateVersionOutputsList, error) {
	return w.svo, nil
}

type testWorkspaceOutputCommandOpts struct {
	items []*tfe.StateVersionOutput
}

func testWorkspaceOutputCommand(t *testing.T, opts *testWorkspaceOutputCommandOpts) (*cli.MockUi, *WorkspaceOutputCommand) {
	t.Helper()

	if opts.items == nil || len(opts.items) < 1 {
		opts.items = []*tfe.StateVersionOutput{
			{
				Name:      "test",
				Value:     "",
				Sensitive: false,
			},
		}
	}

	cloudMockService := &cloud.Cloud{
		WorkspaceService: &WorkspaceOutputReader{
			svo: &tfe.StateVersionOutputsList{
				Items: opts.items,
			},
		},
	}
	ui := cli.NewMockUi()
	meta := NewMeta(cloudMockService)
	meta.Ui = ui
	meta.Env = &environment.CI{}

	return ui, &WorkspaceOutputCommand{Meta: meta}
}

func TestWorkspaceOutputListCommand_Output(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		svoList []*tfe.StateVersionOutput
	}{
		{
			name: "standard-values",
			args: []string{"--workspace=my-workspace"},
			svoList: []*tfe.StateVersionOutput{
				{
					Name:  "image_id",
					Value: "ami-123456",
				},
			},
		},
		{
			name: "sensitive-values",
			args: []string{"--workspace=my-workspace"},
			svoList: []*tfe.StateVersionOutput{
				{
					Name:      "db_creds",
					Value:     "null",
					Sensitive: true,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ui, cmd := testWorkspaceOutputCommand(t, &testWorkspaceOutputCommandOpts{
				items: tc.svoList,
			})

			code := cmd.Run(tc.args)
			if code != 0 {
				t.Fatalf("expected %d but received %d", 0, code)
			}

			stderr := ui.ErrorWriter.String()
			if stderr != "" {
				t.Fatalf("expected %q but received %q", "", stderr)
			}

			stdout := ui.OutputWriter.String()

			var outputVal struct {
				Outputs []WorkspaceOutput `json:"outputs"`
				Status  string            `json:"status"`
			}
			json.Unmarshal([]byte(stdout), &outputVal)

			for i, o := range outputVal.Outputs {
				actualVal, _ := json.Marshal(o.Value)
				expectVal, _ := json.Marshal(tc.svoList[i].Value)
				if !strings.Contains(string(actualVal), string(expectVal)) {
					t.Fatalf("expected %q but received %q", string(expectVal), string(actualVal))
				}
			}
		})
	}
}

func TestWorkspaceOutputListCommand_SuccessArgs(t *testing.T) {
	testCases := []struct {
		name       string
		args       []string
		exitStatus int
	}{
		{
			name:       "valid-args",
			args:       []string{"--workspace", "my-workspace"},
			exitStatus: 0,
		},
		{
			name:       "valid-args-equal",
			args:       []string{"--workspace=my-workspace"},
			exitStatus: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			_, cmd := testWorkspaceOutputCommand(t, &testWorkspaceOutputCommandOpts{})

			if actual := cmd.Run(tc.args); actual != tc.exitStatus {
				t.Errorf("WorkspaceOutputs (%s), expected: %v, actual: %v", tc.name, tc.exitStatus, actual)
			}
		})
	}
}

func TestWorkspaceOutputListCommand_ErrorArgs(t *testing.T) {
	testCases := []struct {
		name         string
		args         []string
		errorMessage string
	}{
		{
			name:         "invalid-args",
			args:         []string{"-workspace"},
			errorMessage: "error parsing command-line flags: flag needs an argument: -workspace",
		},
		{
			name:         "no-args",
			args:         []string{""},
			errorMessage: "error workspace output list requires a workspace name",
		},
		{
			name:         "supported-and-unsupported-args",
			args:         []string{"--workspace=my-workspace", "--unknown"},
			errorMessage: "error parsing command-line flags: flag provided but not defined: -unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			ui, cmd := testWorkspaceOutputCommand(t, &testWorkspaceOutputCommandOpts{})
			// run cmd
			cmd.Run(tc.args)

			output := ui.OutputWriter.String() + ui.ErrorWriter.String()

			if !strings.Contains(output, tc.errorMessage) {
				t.Errorf("expected %q but received %q", tc.errorMessage, output)
			}
		})
	}
}
