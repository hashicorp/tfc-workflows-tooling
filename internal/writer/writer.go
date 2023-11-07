// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package writer

import (
	"log"

	"github.com/mitchellh/cli"
)

type Writer struct {
	json bool
	ui   cli.Ui
}

func NewWriter(ui cli.Ui) *Writer {
	return &Writer{
		ui: ui,
	}
}

func (w *Writer) SetOptions(jsonFlag bool) {
	w.json = jsonFlag
}

// In-Progress diagnostic information
// if *json is set to true, will send log fo to stderr
func (w *Writer) Output(message string) {
	if w.json {
		log.Printf("[INFO] %s", message)
		return
	}

	w.ui.Output(message)
}

// Diagnostic error information
// if *json is set to true, will use log formatting to stderr
func (w *Writer) Error(message string) {
	if w.json {
		log.Printf("[ERROR] %s", message)
		return
	}

	w.ui.Error(message)
}

// Final message sent to stdout stream
// regardless of `json` field we will output the message to stdout stream
// requires the message string is formatted prior to passing to this method receiver
func (w *Writer) OutputResult(message string) {
	w.ui.Output(message)
}

// Final message sent to stderr stream
func (w *Writer) ErrorResult(message string) {
	w.ui.Error(message)
}
