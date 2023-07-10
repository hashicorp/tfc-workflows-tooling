// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-tfe"
	"github.com/sethvargo/go-retry"
)

type WorkspaceService interface {
	ReadStateOutputs(context.Context, string, string) (*tfe.StateVersionOutputsList, error)
}

type workspaceService struct {
	tfe *tfe.Client
}

func wServiceBackoff() retry.Backoff {
	backoff := retry.NewFibonacci(2 * time.Second)
	backoff = retry.WithCappedDuration(7*time.Second, backoff)
	backoff = retry.WithMaxDuration(5*time.Minute, backoff)
	return backoff
}

func (s *workspaceService) ReadStateOutputs(ctx context.Context, orgName string, wName string) (*tfe.StateVersionOutputsList, error) {
	w, wErr := s.tfe.Workspaces.Read(ctx, orgName, wName)
	if wErr != nil {
		return nil, wErr
	}

	if w.ID == "" {
		return nil, fmt.Errorf("unable to find workspace by provided name: '%s'", wName)
	}

	currentSV, csvErr := s.tfe.StateVersions.ReadCurrent(ctx, w.ID)
	if csvErr != nil {
		return nil, csvErr
	}

	// if current state version hasn;t been processed yet,
	// retry/wait until processed or timeout
	if !currentSV.ResourcesProcessed {
		//implement retry
		retryErr := retry.Do(ctx, wServiceBackoff(), func(ctx context.Context) error {
			currentSV, csvErr = s.tfe.StateVersions.ReadCurrent(ctx, w.ID)
			if currentSV.ResourcesProcessed {
				return nil
			}
			if csvErr != nil {
				return csvErr
			}
			return retryableTimeoutError("workspace output list")
		})

		if retryErr != nil {
			return nil, retryErr
		}
	}

	svoList, svoErr := s.tfe.StateVersionOutputs.ReadCurrent(ctx, w.ID)
	if svoErr != nil {
		return nil, svoErr
	}

	return svoList, svoErr
}

func NewWorkspaceService(tfe *tfe.Client) *workspaceService {
	return &workspaceService{tfe}
}
