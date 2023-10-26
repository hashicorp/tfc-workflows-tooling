// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"flag"
	"log"
	"os"

	"github.com/hashicorp/tfci/internal/cloud"
	"github.com/hashicorp/tfci/internal/writer"
	"github.com/hashicorp/tfci/version"

	cmd "github.com/hashicorp/tfci/internal/command"
	"github.com/mitchellh/cli"
)

var (
	hostnameFlag     = flag.String("hostname", "", "The hostname of a Terraform Enterprise installation, if using Terraform Enterprise. Defaults to Terraform Cloud (app.terraform.io)")
	tokenFlag        = flag.String("token", "", "The token used to authenticate with Terraform Cloud. Defaults to reading `TF_API_TOKEN` environment variable")
	organizationFlag = flag.String("organization", "", "Terraform Cloud Organization Name")
	json             = flag.Bool("json", false, "")
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

	// global check if --json flag has been passed
	// otherwise can't determine flag until subcommand Run()s
	for _, arg := range newArgs {
		if arg == "-json" || arg == "--json" {
			*json = true
			continue
		}
	}

	writer := writer.NewWriter(Ui, *json)
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

	cloudService := cloud.NewCloud(tfe, writer)

	meta := cmd.NewMetaOpts(
		appCtx,
		cloudService,
		env,
		cmd.WithOrg(*organizationFlag),
		cmd.WithUi(Ui),
	)

	cliRunner.Commands = map[string]cli.CommandFactory{
		"upload": func() (cli.Command, error) {
			return &cmd.UploadConfigurationCommand{Meta: meta}, nil
		},
		"run create": func() (cli.Command, error) {
			return &cmd.CreateRunCommand{Meta: meta}, nil
		},
		"run apply": func() (cli.Command, error) {
			return &cmd.ApplyRunCommand{Meta: meta}, nil
		},
		"run show": func() (cli.Command, error) {
			return &cmd.ShowRunCommand{Meta: meta}, nil
		},
		"run discard": func() (cli.Command, error) {
			return &cmd.DiscardRunCommand{Meta: meta}, nil
		},
		"run cancel": func() (cli.Command, error) {
			return &cmd.CancelRunCommand{Meta: meta}, nil
		},
		"plan output": func() (cli.Command, error) {
			return &cmd.OutputPlanCommand{Meta: meta}, nil
		},
		"workspace output list": func() (cli.Command, error) {
			return &cmd.WorkspaceOutputCommand{Meta: meta}, nil
		},
	}

	return cliRunner, nil
}
