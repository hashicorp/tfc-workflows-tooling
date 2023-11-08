// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-tfe"
	"github.com/sethvargo/go-retry"
)

type UploadOptions struct {
	Organization           string
	Workspace              string
	ConfigurationDirectory string
	Speculative            bool
}

type ConfigVersionService interface {
	UploadConfig(ctx context.Context, options UploadOptions) (*tfe.ConfigurationVersion, error)
}

type configVersionService struct {
	*cloudMeta
}

func (service *configVersionService) UploadConfig(ctx context.Context, options UploadOptions) (*tfe.ConfigurationVersion, error) {
	workspace, wErr := service.tfe.Workspaces.Read(ctx, options.Organization, options.Workspace)

	if wErr != nil {
		log.Printf("[ERROR] error reading workspace: %q organization: %q error: %s", options.Workspace, options.Organization, wErr)
		return nil, wErr
	}

	configVersion, cvErr := service.tfe.ConfigurationVersions.Create(ctx, workspace.ID, tfe.ConfigurationVersionCreateOptions{
		Speculative:   &options.Speculative,
		AutoQueueRuns: tfe.Bool(false),
	})

	if cvErr != nil {
		log.Printf("[ERROR] error creating configuration version: %s", cvErr)
		return configVersion, cvErr
	}

	service.writer.Output(fmt.Sprintf("Configuration Version has been created: %s", configVersion.ID))

	err := service.tfe.ConfigurationVersions.Upload(ctx, configVersion.UploadURL, options.ConfigurationDirectory)

	if err != nil {
		log.Printf("[ERROR] error uploading configuration version: %s", err)
		return configVersion, err
	}

	service.writer.Output("Uploading configuration...")

	retryErr := retry.Do(ctx, defaultBackoff(), func(ctx context.Context) error {
		log.Printf("[DEBUG] Monitoring Upload Status...")
		cv, err := service.tfe.ConfigurationVersions.Read(ctx, configVersion.ID)
		if err != nil {
			return err
		}
		service.writer.Output(fmt.Sprintf("Upload Status: %q", cv.Status))
		if cv.Status == tfe.ConfigurationUploaded || cv.Status == tfe.ConfigurationErrored {
			// update configVersion to latest results
			configVersion = cv
			return nil
		}
		return retryableTimeoutError("upload configuration")
	})

	if retryErr != nil {
		log.Printf("[ERROR] error waiting for upload completion: %s", retryErr)
		return configVersion, retryErr
	}

	return configVersion, err
}

func NewConfigVersionService(meta *cloudMeta) ConfigVersionService {
	return &configVersionService{meta}
}
