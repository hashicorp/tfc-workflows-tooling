// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package environment

import (
	"os"
	"strconv"
	"sync"
)

type PlatformType string

const (
	GitLab PlatformType = "GitLab"
	GitHub PlatformType = "GitHub"
	Other  PlatformType = "Other"
)

var (
	once   sync.Once
	envCtx CI
)

type GetEnv func(k string) string

type CI struct {
	CI           bool
	PlatformType PlatformType
	Context      Common

	getenv GetEnv
}

type OutputI interface {
	MultiLine() bool
	Value() string
}

type OutputMap map[string]OutputI

func NewOutputMap() OutputMap {
	return OutputMap{}
}

type Common interface {
	ID() string
	SHA() string
	SHAShort() string
	Author() string
	WriteDir() string // where to store tmp files
	SetOutput(output OutputMap)
	CloseOutput() error
}

func (c *CI) initialize() {
	ci, _ := strconv.ParseBool(c.getenv("CI"))
	c.CI = ci
	if c.getenv("GITHUB_ACTIONS") == "true" {
		c.PlatformType = GitHub
		c.Context = newGitHubContext(c.getenv)
		return
	}

	if c.getenv("GITLAB_CI") == "true" {
		c.PlatformType = GitLab
		c.Context = newGitLabContext(c.getenv)
		return
	}

	c.PlatformType = Other
}

func NewCIContext() *CI {
	once.Do(func() {
		envCtx = CI{
			getenv: os.Getenv,
		}
		envCtx.initialize()
	})
	return &envCtx
}
