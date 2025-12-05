// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"fmt"

	"github.com/hashicorp/go-tfe"
)

type Writer interface {
	UseJson(json bool)
	Output(msg string)
	Error(msg string)
}

type defaultWriter struct{}

func (d *defaultWriter) UseJson(json bool) {}
func (d *defaultWriter) Output(msg string) {}
func (d *defaultWriter) Error(msg string)  {}

// compile time check
var _ Writer = (*defaultWriter)(nil)

type Cloud struct {
	*cloudMeta

	ConfigVersionService
	RunService
	PlanService
	WorkspaceService
	PolicyService
}

func (c *Cloud) UseJson(json bool) {
	c.writer.UseJson(json)
}

// RunLinkByID constructs a run link URL using only the run ID.
// This is useful when we don't have the full Run object (e.g., from policy operations).
func (c *Cloud) RunLinkByID(organization, runID string) string {
	url := c.tfe.BaseURL()
	return fmt.Sprintf("%s://%s/app/%s/runs/%s", url.Scheme, url.Host, organization, runID)
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
		PolicyService:        NewPolicyService(meta),
	}
}
