// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package environment

import (
	"fmt"
	"os"
	"strings"
)

const EOF = "\n"

// Sourced from: https://docs.github.com/en/actions/learn-github-actions/variables#default-environment-variables
type GitHubContext struct {
	// A unique number for each workflow run within a repository. This number does not change if you re-run the workflow run
	runId string
	// A unique number for each run of a particular workflow in a repository. This number begins at 1 for the workflow's first run, and increments with each new run. This number does not change if you re-run the workflow run
	runNumber string
	// The commit SHA that triggered the workflow. The value of this commit SHA depends on the event that triggered the workflow.
	commitSHA string
	// The name of the person or app that initiated the workflow. For example, octocat.
	actor string
	// The owner and repository name. For example, octocat/Hello-World
	repository string
	// The short ref name of the branch or tag that triggered the workflow run. This value matches the branch or tag name shown on GitHub
	refName string
	// The type of ref that triggered the workflow run. Valid values are branch or tag.
	refType string
	// The path to a temporary directory on the runner. This directory is emptied at the beginning and end of each job. Note that files will not be removed if the runner's user account does not have permission to delete them.
	runnerTemp string
	// path to ::set-output
	githubOutput string
	// data sent to GITHUB_OUTPUT
	output OutputMap
	//
	fileDelimeter string
}

func (gh *GitHubContext) ID() string {
	return fmt.Sprintf("gha-%s-%s", gh.runId, gh.runNumber)
}

func (gh *GitHubContext) SHA() string {
	return gh.commitSHA
}
func (gh *GitHubContext) SHAShort() string {
	if len(gh.commitSHA) > 7 {
		return gh.commitSHA[:7]
	}
	return gh.commitSHA
}

func (gh *GitHubContext) Author() string {
	return gh.actor
}

func (gh *GitHubContext) WriteDir() string {
	return gh.runnerTemp
}

func (gh *GitHubContext) SetOutput(output OutputMap) {
	gh.output = output
}

func (gh *GitHubContext) CloseOutput() (retErr error) {
	filepath := gh.githubOutput
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		retErr = err
		return
	}

	data := []string{}
	for k, v := range gh.output {
		data = append(data, multiLineStrVal(gh.fileDelimeter, k, v.String()))
	}
	out := []byte(strings.Join(data, EOF))

	defer func() {
		if err := file.Close(); err != nil {
			retErr = err
		}
	}()

	if _, err := file.Write(out); err != nil {
		retErr = err
		return
	}

	// reset output
	gh.output = make(map[string]OutputWriter)

	return
}

func newGitHubContext(getenv GetEnv) *GitHubContext {
	ghCtx := &GitHubContext{
		runId:        getenv("GITHUB_RUN_ID"),
		runNumber:    getenv("GITHUB_RUN_NUMBER"),
		commitSHA:    getenv("GITHUB_SHA"),
		actor:        getenv("GITHUB_ACTOR"),
		repository:   getenv("GITHUB_REPOSITORY"),
		refName:      getenv("GITHUB_REF_NAME"),
		refType:      getenv("GITHUB_REF_TYPE"),
		githubOutput: getenv("GITHUB_OUTPUT"),
		runnerTemp:   getenv("RUNNER_TEMP"),
		output:       make(map[string]OutputWriter),
	}
	// set random/unique to each github action runner
	ghCtx.fileDelimeter = fmt.Sprintf("_GH%s%sFD_", ghCtx.runId, ghCtx.runNumber)
	return ghCtx
}

func multiLineStrVal(fileD, k, v string) string {
	return fmt.Sprintf("%s<<"+fileD+EOF+"%s"+EOF+fileD, k, v)
}
