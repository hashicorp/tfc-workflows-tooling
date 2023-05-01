// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"

	"github.com/hashicorp/tfci/internal/cloud"
	"github.com/hashicorp/tfci/internal/environment"
	"github.com/mitchellh/cli"
)

type Status string

const (
	Success Status = "Success"
	Error   Status = "Error"
	Timeout Status = "Timeout"
	Noop    Status = "Noop"
)

type Meta struct {
	// Organization for Terraform Cloud installation
	Organization string
	// cli ui settings
	Ui cli.Ui
	// parent context
	Context context.Context
	// CI environment variables & output
	Env *environment.CI
	// shared go-tfe client
	cloud *cloud.Cloud
	// if github/gitlab context is not detected
	cliOut map[string]string
}

func (c *Meta) flagSet(name string) *flag.FlagSet {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(ioutil.Discard)
	f.Usage = func() {}

	return f
}

func (c *Meta) resolveStatus(err error) Status {
	if err != nil {
		switch err.(type) {
		case *cloud.RetryTimeoutError:
			return Timeout
		default:
			return Error
		}
	}
	return Success
}

func (c *Meta) addOutput(key string, value string) {
	// check if context in not github/gitlab
	if c.Env.Context != nil {
		c.Env.Context.AddOutput(key, value)
	} else {
		// cli use
		c.cliOut[key] = value
	}
}

func (c *Meta) closeOutput() string {
	// check if context in not github/gitlab
	var m map[string]string
	if c.Env.Context != nil {
		m = c.Env.Context.GetMessages()
		c.Env.Context.CloseOutput()
	} else {
		m = c.cliOut
	}

	// remove from stdout
	delete(m, "payload")

	outJson, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return string(err.Error())
	}
	return string(outJson)
}

func NewMeta(c *cloud.Cloud) *Meta {
	return &Meta{
		cloud:  c,
		cliOut: make(map[string]string),
	}
}
