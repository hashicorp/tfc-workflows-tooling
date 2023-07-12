// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

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
	messages map[string]*outputMessage
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
	c.messages[name] = newOutputMessage(name, value, defaultOutputOpts)
}

// adds new output value with options &outputOpts{}
func (c *Meta) addOutputWithOpts(name string, value interface{}, opts *outputOpts) {
	c.messages[name] = newOutputMessage(name, value, opts)
}

// returns json result string, containing all outputs
// if running in ci, will send outputs to platform
func (c *Meta) closeOutput() string {
	// using map[string]any to pretty marshal collection
	stdOutput := make(map[string]interface{})
	// map[string]OutputI interface
	platOutput := environment.NewOutputMap()

	for _, m := range c.messages {
		// some values we may want to exclude for stdout
		if m.stdOut {
			stdOutput[m.name] = m.value
		}
		// some outputs we may want to exclude for platform
		if m.IncludeWithPlatform() {
			val, err := m.Value()
			// if error, add to logger
			if err != nil {
				log.Printf("[ERROR] problem writing output: '%s', with: %s", m.name, err.Error())
				// don't include value to platform is issue serializing
				continue
			}
			platOutput[m.name] = environment.NewOutput(val, m.multiLine)
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
		cloud:    c,
		messages: make(map[string]*outputMessage),
	}
}
