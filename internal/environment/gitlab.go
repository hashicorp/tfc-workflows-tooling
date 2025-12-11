// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package environment

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Sourced: from https://docs.gitlab.com/ee/ci/variables/predefined_variables.html
type GitLabContext struct {
	// The unique ID of build execution in a single executor.
	concurrentId string
	// The unique ID of build execution in a single executor and project.
	concurrentProjectId string
	// The name of the job being run
	jobName string
	// The commit revision the project is built for.
	commitSHA string
	//The first eight characters of CI_COMMIT_SHA.
	commitSHAShort string
	// The author of the commit in Name <email> format.
	commitAuthor string
	// The branch or tag name for which project is built.
	commitRefName string
	// The full commit message.
	commitMessage string
	// The map containing output data
	output OutputMap
}

func writeArtifact(prefix string, name string, data string) (err error) {
	file, err := os.Create(generateArtifactFileName("json", prefix, name))
	if err != nil {
		return
	}
	defer func() {
		err = file.Close()
	}()

	_, err = file.WriteString(data)

	return err
}

func (gl *GitLabContext) ID() string {
	return fmt.Sprintf("gl-%s-%s", gl.concurrentId, gl.concurrentProjectId)
}

func (gl *GitLabContext) SHA() string {
	return gl.commitSHA
}

func (gl *GitLabContext) SHAShort() string {
	return gl.commitSHAShort
}

func (gl *GitLabContext) Author() string {
	return gl.commitAuthor
}

func (gh *GitLabContext) WriteDir() string {
	// figure out where to store tmp files on gitlab pipeline runner
	// or let --location= flag dictate
	return ""
}

func (gl *GitLabContext) SetOutput(output OutputMap) {
	gl.output = output
}

func (gl *GitLabContext) CloseOutput() (err error) {
	log.Printf("Gitlab flushing output")

	// Create output file
	file, err := os.Create(".env")
	if err != nil {
		return
	}
	defer func() {
		err = file.Close()
	}()

	var lines []string
	for k, v := range gl.output {
		if v.MultiLine() {
			if err = writeArtifact(gl.jobName, k, v.String()); err != nil {
				return
			}
			continue
		}

		line := fmt.Sprintf("%s=%s", k, v.String())
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	if _, err := file.WriteString(content); err != nil {
		return err
	}

	return
}

func generateArtifactFileName(ext string, parts ...string) string {
	return fmt.Sprintf("%s.%s", strings.Join(parts, "_"), ext)
}

func newGitLabContext(getenv GetEnv) *GitLabContext {
	return &GitLabContext{
		concurrentId:        getenv("CI_CONCURRENT_ID"),
		concurrentProjectId: getenv("CI_CONCURRENT_PROJECT_ID"),
		jobName:             getenv("CI_JOB_NAME"),
		commitSHA:           getenv("CI_COMMIT_SHA"),
		commitSHAShort:      getenv("CI_COMMIT_SHORT_SHA"),
		commitAuthor:        getenv("CI_COMMIT_AUTHOR"),
		commitMessage:       getenv("CI_COMMIT_MESSAGE"),
		commitRefName:       getenv("CI_COMMIT_REF_NAME"),
		output:              make(map[string]OutputWriter),
	}
}
