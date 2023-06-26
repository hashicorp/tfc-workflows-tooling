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

func (c *Meta) addOutput(name string, value string) {
	c.messageOutput[name] = newOutputMessage(name, value)
}

type outputOpts struct {
	stdOut    bool
	multiLine bool
}

func (c *Meta) addOutputWithOpts(name string, value interface{}, opts *outputOpts) {
	msg := newOutputMessage(name, value)
	msg.stdOut = opts.stdOut
	msg.multiLine = opts.multiLine
	c.messageOutput[name] = msg
}

type PlatformOutput struct {
	value     string
	multiLine bool
}

func (p *PlatformOutput) Value() string {
	return p.value
}

func (p *PlatformOutput) MultiLine() bool {
	return p.multiLine
}

func (c *Meta) closeOutput() string {
	stdOutput := make(map[string]interface{})
	platOutput := environment.NewOutputMap()

	for _, m := range c.messageOutput {
		if m.stdOut {
			stdOutput[m.name] = m.value
		}
		if m.IncludeWithPlatform() {
			platOutput[m.name] = &PlatformOutput{
				value:     m.String(),
				multiLine: m.multiLine,
			}
		}
	}

	if c.Env.Context != nil {
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
