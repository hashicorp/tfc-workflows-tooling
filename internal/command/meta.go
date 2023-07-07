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
	// messages for stdout, platform output
	messageOutput map[string]*OutputMessage
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

// adds new output value to map as &OutputMessage{}
func (c *Meta) addOutput(name string, value string) {
	c.messageOutput[name] = newOutputMessage(name, value)
}

type outputOpts struct {
	// indicates if value should be displayed to stdout
	stdOut bool
	// indicates if value contains a multiline value as some platforms: gitlab do not support multiline values in `.env`
	multiLine bool
}

// adds new output value with options &outputOpts{}
func (c *Meta) addOutputWithOpts(name string, value interface{}, opts *outputOpts) {
	msg := newOutputMessage(name, value)
	msg.stdOut = opts.stdOut
	msg.multiLine = opts.multiLine
	c.messageOutput[name] = msg
}

// returns json result string, containing all outputs
// if running in ci, will send outputs to platform
func (c *Meta) closeOutput() string {
	// using map[string]any to pretty marshal collection
	stdOutput := make(map[string]interface{})
	// map[string]OutputI interface
	platOutput := environment.NewOutputMap()

	for _, m := range c.messageOutput {
		// some values we may want to exclude for stdout
		if m.stdOut {
			stdOutput[m.name] = m.value
		}
		// some outputs we may want to exclude for platform
		if m.IncludeWithPlatform() {
			platOutput[m.name] = m
		}
	}

	// check to see if we're running in CI environment
	if c.Env.Context != nil {
		// pass output data and close signifying we're done
		c.Env.Context.SetOutput(platOutput)
		c.Env.Context.CloseOutput()
	}

	outJson, err := json.MarshalIndent(stdOutput, "", "  ")
	if err != nil {
		return string(err.Error())
	}
	return string(outJson)
}

func NewMeta(c *cloud.Cloud) *Meta {
	return &Meta{
		cloud:         c,
		messageOutput: make(map[string]*OutputMessage),
	}
}
