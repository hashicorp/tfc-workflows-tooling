package writer

import (
	"log"

	"github.com/mitchellh/cli"
)

type Writer struct {
	json bool
	ui   cli.Ui
}

func NewWriter(ui cli.Ui, jsonFlag bool) *Writer {
	return &Writer{
		json: jsonFlag,
		ui:   ui,
	}
}

func (w *Writer) Output(message string) {
	if w.json {
		log.Printf("[INFO] %s", message)
		return
	}

	w.ui.Output(message)
}

func (w *Writer) Error(message string) {
	if w.json {
		log.Printf("[ERROR] %s", message)
		return
	}

	w.ui.Error(message)
}
