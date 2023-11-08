// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/hashicorp/tfci/internal/cloud"
	"github.com/hashicorp/tfci/internal/environment"
)

type Status string

const (
	Success Status = "Success"
	Error   Status = "Error"
	Timeout Status = "Timeout"
	Noop    Status = "Noop"
)

type Writer interface {
	UseJson(json bool)
	Output(msg string)
	Error(msg string)
	OutputResult(msg string)
	ErrorResult(msg string)
}

type Meta struct {
	// Organization for Terraform Cloud installation
	organization string
	// parent context
	appCtx context.Context
	// CI environment variables & output
	env *environment.CI
	// shared go-tfe client
	cloud *cloud.Cloud
	// messages for stdout, platform output
	messages map[string]*outputMessage
	// writer interface to handle result and diagnostic information
	writer Writer
	// flag to prevent non-json messages to stdout
	json bool
}

func (c *Meta) setupCmd(args []string, flags *flag.FlagSet) error {
	if err := flags.Parse(args); err != nil {
		c.emitFlagOptions()
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.writer.ErrorResult(fmt.Sprintf("error parsing command-line flags: %s\n", err.Error()))
		return err
	}

	c.emitFlagOptions()
	return nil
}

func (c *Meta) flagSet(name string) *flag.FlagSet {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(ioutil.Discard)
	f.Usage = func() {}

	f.BoolVar(&c.json, "json", false, "Suppresses all logs and instead returns output value in JSON format")

	return f
}

func (c *Meta) emitFlagOptions() {
	// configure json option for command writer
	c.writer.UseJson(c.json)
	// configure json option for cloud writer
	c.cloud.UseJson(c.json)
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
			// add raw interface{} value to stdout
			stdOutput[m.name] = m.value
		}
		// some outputs we may want to exclude for platform
		if m.IncludeWithPlatform() {
			// convert to string
			val, err := m.Value()
			// if error, add to logger
			if err != nil {
				log.Printf("[ERROR] problem writing output: '%s', with: %s", m.name, err.Error())
				// don't include value if issue serializing value
				continue
			}
			platOutput[m.name] = environment.NewOutput(val, m.multiLine)
		}
	}

	// check to see if we're running in CI environment
	if c.env.Context != nil {
		// pass output data and close signifying we're done
		c.env.Context.SetOutput(platOutput)
		c.env.Context.CloseOutput()
	}

	outJson, err := json.MarshalIndent(stdOutput, "", "  ")
	if err != nil {
		return string(err.Error())
	}
	return string(outJson)
}

func WithOrg(org string) func(*Meta) {
	return func(m *Meta) {
		m.organization = org
	}
}

func WithWriter(w Writer) func(*Meta) {
	return func(m *Meta) {
		m.writer = w
	}
}

func NewMetaOpts(ctx context.Context, tfeClient *cloud.Cloud, ciEnv *environment.CI, setters ...func(*Meta)) *Meta {
	m := &Meta{
		cloud:    tfeClient,
		appCtx:   ctx,
		env:      ciEnv,
		messages: make(map[string]*outputMessage),
	}

	for _, setter := range setters {
		setter(m)
	}

	return m
}

func NewMeta(c *cloud.Cloud) *Meta {
	return &Meta{
		cloud:    c,
		messages: make(map[string]*outputMessage),
	}
}
