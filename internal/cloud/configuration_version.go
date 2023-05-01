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
	*tfe.Client
}

func (service *configVersionService) UploadConfig(ctx context.Context, options UploadOptions) (*tfe.ConfigurationVersion, error) {
	workspace, wErr := service.Workspaces.Read(ctx, options.Organization, options.Workspace)

	if wErr != nil {
		return nil, wErr
	}

	configVersion, cvErr := service.ConfigurationVersions.Create(ctx, workspace.ID, tfe.ConfigurationVersionCreateOptions{
		Speculative:   &options.Speculative,
		AutoQueueRuns: tfe.Bool(false),
	})

	if cvErr != nil {
		return configVersion, cvErr
	}

	fmt.Printf("Configuration Version has been created: %s\n", configVersion.ID)

	err := service.ConfigurationVersions.Upload(ctx, configVersion.UploadURL, options.ConfigurationDirectory)

	if err != nil {
		return configVersion, err
	}

	fmt.Println("Uploading configuration...")
	retryErr := retry.Do(ctx, defaultBackoff(), func(ctx context.Context) error {
		log.Printf("[DEBUG] Monitoring Upload Status...")
		cv, err := service.ConfigurationVersions.Read(ctx, configVersion.ID)
		if err != nil {
			return err
		}
		fmt.Printf("Upload Status: '%s'\n", cv.Status)
		if cv.Status == tfe.ConfigurationUploaded || cv.Status == tfe.ConfigurationErrored {
			// update configVersion to latest results
			configVersion = cv
			return nil
		}
		return retryableTimeoutError("upload configuration")
	})

	if retryErr != nil {
		return configVersion, retryErr
	}

	return configVersion, err
}

func NewConfigVersionService(c *tfe.Client) ConfigVersionService {
	return &configVersionService{c}
}
