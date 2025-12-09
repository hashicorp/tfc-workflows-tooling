// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package logging

import (
	"io"
	"log"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/tfci/internal/environment"
)

const envLogLevel = "TF_LOG"

var (
	ValidLevels = []string{"DEBUG", "INFO", "WARN", "ERROR"}
	logger      hclog.Logger
	logWriter   io.Writer
)

type LoggerOptions struct {
	PlatformType environment.PlatformType
}

func SetupLogger(options *LoggerOptions) {
	logLevel := os.Getenv(envLogLevel)
	// default to off
	if logLevel == "" {
		logLevel = "OFF"
	}
	logger = hclog.NewInterceptLogger(&hclog.LoggerOptions{
		Name:  "tfci",
		Level: hclog.LevelFromString(logLevel),
	})
	logger.With("platform", options.PlatformType)
	logWriter = logger.StandardWriter(&hclog.StandardLoggerOptions{InferLevels: true})

	// set up the default std library logger to use our output
	log.SetFlags(0)
	log.SetPrefix("")
	log.SetOutput(logWriter)
}
