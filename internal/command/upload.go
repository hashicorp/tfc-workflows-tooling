// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/tfci/internal/cloud"
)

type UploadConfigurationCommand struct {
	*Meta
	Workspace   string
	Directory   string
	Speculative bool
	Provisional bool
}

func (c *UploadConfigurationCommand) flags() *flag.FlagSet {
	f := c.flagSet("upload")

	f.StringVar(&c.Workspace, "workspace", "", "The name of the workspace to create the new configuration version in.")
	f.StringVar(&c.Directory, "directory", "", "Path to the configuration files on disk.")
	f.BoolVar(&c.Speculative, "speculative", false, "When true, this configuration version may only be used to create runs which are speculative, that is, can neither be confirmed nor applied.")
	f.BoolVar(&c.Provisional, "provisional", false, "When true, this configuration version does not immediately become the workspace's current configuration until a run referencing it is ultimately applied.")
	return f
}

func (c *UploadConfigurationCommand) Run(args []string) int {
	if err := c.setupCmd(args, c.flags()); err != nil {
		return 1
	}

	log.Printf("[DEBUG] uploading configuration with, workspace: %s, directory: %s, speculative: %t, provisional: %t", c.Workspace, c.Directory, c.Speculative, c.Provisional)

	dirPath, dirError := filepath.Abs(c.Directory)
	if dirError != nil {
		c.addOutput("status", string(Error))
		c.closeOutput()
		c.writer.ErrorResult(fmt.Sprintf("error resolving directory path %s", dirError.Error()))
		return 1
	}

	log.Printf("[DEBUG] target directory for configuration upload: %s", dirPath)

	configVersion, cvError := c.cloud.UploadConfig(c.appCtx, cloud.UploadOptions{
		Workspace:              c.Workspace,
		Organization:           c.organization,
		ConfigurationDirectory: dirPath,
		Speculative:            c.Speculative,
		Provisional:            c.Provisional,
	})

	if cvError != nil {
		status := c.resolveStatus(cvError)
		c.addOutput("status", string(status))
		c.addConfigurationDetails(configVersion)
		c.writer.ErrorResult(fmt.Sprintf("error uploading configuration version to Terraform Cloud: %s", cvError.Error()))
		c.writer.OutputResult(c.closeOutput())
		return 1
	}

	c.addOutput("status", string(Success))
	c.addConfigurationDetails(configVersion)
	c.writer.OutputResult(c.closeOutput())
	return 0
}

func (c *UploadConfigurationCommand) addConfigurationDetails(config *tfe.ConfigurationVersion) {
	if config != nil {
		c.addOutput("configuration_version_id", config.ID)
		c.addOutput("configuration_version_status", string(config.Status))
	}

	c.addOutputWithOpts("payload", config, &outputOpts{
		stdOut:      false,
		multiLine:   true,
		platformOut: true,
	})
}

func (c *UploadConfigurationCommand) Help() string {
	helpText := `
Usage: tfci [global options] upload [options]

	Creates and uploads a new configuration version for the provided workspace.

Global Options:

	-hostname       The hostname of a Terraform Enterprise installation, if using Terraform Enterprise. Defaults to "app.terraform.io".

	-token          The token used to authenticate with Terraform Cloud. Defaults to reading "TF_API_TOKEN" environment variable.

	-organization   Terraform Cloud Organization Name.

Options:

	-workspace      The name of the Terraform Cloud Workspace to create and upload the terraform configuration version in.

	-directory      Path to the terraform configuration files on disk.

	-speculative    When true, this configuration version may only be used to create runs which are speculative, that is, can neither be confirmed nor applied.

	-provisional    When true, this configuration version does not immediately become the workspace's current configuration until a run referencing it is ultimately applied.
	`
	return strings.TrimSpace(helpText)
}

func (c *UploadConfigurationCommand) Synopsis() string {
	return "Creates and uploads a new configuration version for the provided workspace"
}
