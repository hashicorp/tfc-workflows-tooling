// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"

	"github.com/hashicorp/go-tfe"
)

type PlanService interface {
	GetPlan(context.Context, string) (*tfe.Plan, error)
}

type planService struct {
	tfe *tfe.Client
}

func (service *planService) GetPlan(ctx context.Context, planID string) (*tfe.Plan, error) {
	data, err := service.tfe.Plans.Read(ctx, planID)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func NewPlanService(tfe *tfe.Client) *planService {
	return &planService{tfe}
}
