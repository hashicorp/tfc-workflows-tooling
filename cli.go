// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"flag"
	"log"
	"os"

	"github.com/hashicorp/tfci/internal/cloud"
	"github.com/hashicorp/tfci/version"

	"github.com/hashicorp/tfci/internal/command"
	"github.com/mitchellh/cli"
)

var (
	hostnameFlag     = flag.String("hostname", "", "The hostname of a Terraform Enterprise installation, if using Terraform Enterprise. Defaults to Terraform Cloud (app.terraform.io)")
	tokenFlag        = flag.String("token", "", "The token used to authenticate with Terraform Cloud. Defaults to reading `TF_API_TOKEN` environment variable")
	organizationFlag = flag.String("organization", "", "Terraform Cloud Organization Name")
)

func newCliRunner() (*cli.CLI, error) {
	args := os.Args[1:]
	log.Printf("[DEBUG] Command argument count: %d", len(args))

	err := flag.CommandLine.Parse(args)
	if err != nil {
		return nil, err
	}

	newArgs := flag.CommandLine.Args()

	cliRunner := cli.NewCLI("tfc", version.GetVersion())
	cliRunner.Args = newArgs

	orgEnv := os.Getenv("TF_CLOUD_ORGANIZATION")

	if *organizationFlag == "" && orgEnv != "" {
		*organizationFlag = orgEnv
	}
	log.Printf("[DEBUG] Subcommand arg count: %d for organization: %s", len(newArgs), orgEnv)

	tfe, err := cloud.NewTfeClient(*hostnameFlag, *tokenFlag, string(env.PlatformType))
	if err != nil {
		log.Printf("[ERROR] Could not initialize terraform cloud client, error: %#v", err)
		return nil, err
	}

	c := cloud.NewCloud(tfe)

	meta := command.NewMeta(c)
	meta.Ui = Ui
	meta.Organization = *organizationFlag
	meta.Context = appCtx
	meta.Env = env

	cliRunner.Commands = map[string]cli.CommandFactory{
		"upload": func() (cli.Command, error) {
			return &command.UploadConfigurationCommand{Meta: meta}, nil
		},
		"run create": func() (cli.Command, error) {
			return &command.CreateRunCommand{Meta: meta}, nil
		},
		"run apply": func() (cli.Command, error) {
			return &command.ApplyRunCommand{Meta: meta}, nil
		},
		"run show": func() (cli.Command, error) {
			return &command.ShowRunCommand{Meta: meta}, nil
		},
		"run discard": func() (cli.Command, error) {
			return &command.DiscardRunCommand{Meta: meta}, nil
		},
		"run cancel": func() (cli.Command, error) {
			return &command.CancelRunCommand{Meta: meta}, nil
		},
		"plan output": func() (cli.Command, error) {
			return &command.OutputPlanCommand{Meta: meta}, nil
		},
		"workspace output list": func() (cli.Command, error) {
			return &command.WorkspaceOutputCommand{Meta: meta}, nil
		},
	}

	return cliRunner, nil
}
