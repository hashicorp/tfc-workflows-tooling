// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"
	"log"
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

// wait 5 minutes for current state version finish processing
// primarily to prevent edge case of reading workspace outputs immediately after an apply run
const StateVersionOutputMaxDuration = 5 * time.Minute

func wServiceBackoff() retry.Backoff {
	backoff := retry.NewFibonacci(2 * time.Second)
	backoff = retry.WithCappedDuration(7*time.Second, backoff)
	backoff = retry.WithMaxDuration(StateVersionOutputMaxDuration, backoff)
	return backoff
}

func (s *workspaceService) ReadStateOutputs(ctx context.Context, orgName string, wName string) (*tfe.StateVersionOutputsList, error) {
	w, wErr := s.tfe.Workspaces.Read(ctx, orgName, wName)
	if wErr != nil {
		log.Printf("[ERROR] error reading workspace: %q organization: %q, error: %s", wName, orgName, wErr)
		return nil, wErr
	}

	currentSV, csvErr := s.tfe.StateVersions.ReadCurrent(ctx, w.ID)
	if csvErr != nil {
		log.Printf("[ERROR] error reading current state version: %s", csvErr)
		return nil, csvErr
	}

	// if current state version has not been processed yet,
	// poll/wait for current state version to finish processing
	if !currentSV.ResourcesProcessed {
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
			log.Printf("[ERROR] error waiting for current state version to finish processing: %s", retryErr)
			return nil, retryErr
		}
	}

	svoList, svoErr := s.tfe.StateVersionOutputs.ReadCurrent(ctx, w.ID)
	if svoErr != nil {
		log.Printf("[ERROR] error reading state version output list: %s", svoErr)
		return nil, svoErr
	}

	return svoList, svoErr
}

func NewWorkspaceService(tfe *tfe.Client) *workspaceService {
	return &workspaceService{tfe}
}
