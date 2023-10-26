// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"github.com/hashicorp/go-tfe"
)

type Writer interface {
	Output(msg string)
	Error(msg string)
}

type defaultWriter struct{}

func (d *defaultWriter) Output(msg string) {}
func (d *defaultWriter) Error(msg string)  {}

// compile time check
var _ Writer = (*defaultWriter)(nil)

type Cloud struct {
	ConfigVersionService
	RunService
	PlanService
	WorkspaceService
}

func NewCloud(c *tfe.Client, w Writer) *Cloud {
	return &Cloud{
		ConfigVersionService: NewConfigVersionService(c, w),
		RunService:           NewRunService(c, w),
		PlanService:          NewPlanService(c, w),
		WorkspaceService:     NewWorkspaceService(c, w),
	}
}
