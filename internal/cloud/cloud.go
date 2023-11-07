// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"github.com/hashicorp/go-tfe"
)

type Writer interface {
	SetOptions(json bool)
	Output(msg string)
	Error(msg string)
}

type defaultWriter struct{}

func (d *defaultWriter) SetOptions(json bool) {}
func (d *defaultWriter) Output(msg string)    {}
func (d *defaultWriter) Error(msg string)     {}

// compile time check
var _ Writer = (*defaultWriter)(nil)

type Cloud struct {
	*cloudMeta

	ConfigVersionService
	RunService
	PlanService
	WorkspaceService
}

func (c *Cloud) InjectJson(json bool) {
	c.writer.SetOptions(json)
	c.cloudMeta.writer.SetOptions(json)
}

// shared struct to embed
type cloudMeta struct {
	tfe    *tfe.Client
	writer Writer
}

func NewCloud(c *tfe.Client, w Writer) *Cloud {
	meta := &cloudMeta{
		tfe:    c,
		writer: w,
	}

	return &Cloud{
		cloudMeta:            meta,
		ConfigVersionService: NewConfigVersionService(meta),
		RunService:           NewRunService(meta),
		PlanService:          NewPlanService(meta),
		WorkspaceService:     NewWorkspaceService(meta),
	}
}
