// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"github.com/hashicorp/go-tfe"
)

type Cloud struct {
	ConfigVersionService
	RunService
	PlanService
	WorkspaceService
}

func NewCloud(c *tfe.Client) *Cloud {
	return &Cloud{
		ConfigVersionService: NewConfigVersionService(c),
		RunService:           NewRunService(c),
		PlanService:          NewPlanService(c),
		WorkspaceService:     NewWorkspaceService(c),
	}
}
