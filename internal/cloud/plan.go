// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"
	"log"

	"github.com/hashicorp/go-tfe"
)

type PlanService interface {
	GetPlan(context.Context, string) (*tfe.Plan, error)
}

type planService struct {
	tfe *tfe.Client

	writer Writer
}

func (service *planService) GetPlan(ctx context.Context, planID string) (*tfe.Plan, error) {
	data, err := service.tfe.Plans.Read(ctx, planID)
	if err != nil {
		log.Printf("[ERROR] error reading plan: '%s', with: '%s'", planID, err.Error())
		return nil, err
	}
	return data, nil
}

func NewPlanService(tfe *tfe.Client, w Writer) *planService {
	return &planService{tfe, w}
}
