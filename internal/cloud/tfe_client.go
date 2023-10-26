// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/tfci/version"
)

const (
	defaultHostname = "app.terraform.io"
	baseUserAgent   = "tfci"
	unknownPlatform = "other"
)

func getUserAgent(platform string) string {
	var agent string
	platform = strings.ToLower(platform)
	version := version.GetVersion()
	if platform != unknownPlatform {
		agent = fmt.Sprintf("%s/%s %s", baseUserAgent, version, platform)
	} else {
		agent = fmt.Sprintf("%s/%s", baseUserAgent, version)
	}
	return agent
}

func NewTfeClient(hostFlag string, tokenFlag string, platform string) (*tfe.Client, error) {
	tfeConfig := tfe.DefaultConfig()

	host := hostFlag
	if hostFlag == "" {
		hostEnv := os.Getenv("TF_HOSTNAME")
		if hostEnv != "" {
			host = hostEnv
		} else {
			host = defaultHostname
		}
	}

	log.Printf("[DEBUG] Initializing terraform cloud client, host: %s", host)

	token := tokenFlag
	if tokenFlag == "" {
		tokenEnv := os.Getenv("TF_API_TOKEN")
		if tokenEnv != "" {
			token = tokenEnv
		}
	}

	tfeConfig.Headers.Set("User-Agent", getUserAgent(platform))
	tfeConfig.Address = fmt.Sprintf("https://%s", host)
	tfeConfig.Token = token

	if tfeConfig.Token == "" {
		return nil, fmt.Errorf("terraform cloud API token is not set")
	}

	log.Printf("[DEBUG] token has been set")

	client, err := tfe.NewClient(tfeConfig)
	if err != nil {
		return nil, err
	}

	client.RetryServerErrors(true)

	log.Printf("[DEBUG] TFC/E Version: %s", client.RemoteAPIVersion())

	return client, nil
}
