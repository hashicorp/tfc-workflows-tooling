// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"log"
	"os"
	"runtime"

	"github.com/hashicorp/tfci/internal/environment"
	"github.com/hashicorp/tfci/internal/logging"
	"github.com/hashicorp/tfci/version"
	"github.com/mitchellh/cli"
)

var (
	Ui     cli.Ui
	appCtx context.Context
	env    *environment.CI
)

func main() {
	// load env
	env = environment.NewCIContext()

	// setup logging
	logging.SetupLogger(&logging.LoggerOptions{
		PlatformType: env.PlatformType,
	})

	// Ui settings
	Ui = &cli.ColoredUi{
		ErrorColor: cli.UiColorRed,
		WarnColor:  cli.UiColorYellow,
		Ui: &cli.BasicUi{
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
			Reader:      os.Stdin,
		},
	}

	appCtx = context.Background()

	os.Exit(realMain())
}

func realMain() int {
	log.Printf("[INFO] version: %s", version.GetVersion())
	log.Printf("[INFO] Go runtime version: %s", runtime.Version())

	log.Printf("[DEBUG] Preparing runner")
	cliRunner, runError := newCliRunner()
	if runError != nil {
		Ui.Error(runError.Error())
		return 1
	}

	log.Printf("[DEBUG] Running command")
	exitCode, err := cliRunner.Run()
	if err != nil {
		Ui.Error(err.Error())
		return 1
	}

	return exitCode
}
