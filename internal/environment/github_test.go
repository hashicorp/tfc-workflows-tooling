// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package environment

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func randomSha(t *testing.T) string {
	t.Helper()
	timestamp := time.Now().Unix()
	bytes := sha256.Sum256([]byte(fmt.Sprint(timestamp)))
	return fmt.Sprintf("%x", bytes)
}

func getEnvMock(t *testing.T) map[string]string {
	t.Helper()
	return map[string]string{
		"GITHUB_RUN_ID":     "12345",
		"GITHUB_RUN_NUMBER": "1",
		"GITHUB_SHA":        randomSha(t),
		"GITHUB_OUTPUT":     "github_output",
		"RUNNER_TEMP":       "/runner/temp",
	}
}

func createOutFile(t *testing.T, path string) {
	t.Helper()
	var _, err = os.Stat(path)

	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		if err != nil {
			t.Fatalf("error: %s", err)
			return
		}
		defer file.Close()
	}

	t.Cleanup(func() {
		err := os.Remove(path)
		if err != nil {
			t.Fatalf("failed to cleanup file: %s", path)
		}
	})
}

type testOutput struct {
	val       string
	multiLine bool
}

func (o *testOutput) MultiLine() bool {
	return o.multiLine
}

func (o *testOutput) Value() string {
	return o.val
}

func Test_GitHubOutput(t *testing.T) {
	env := getEnvMock(t)
	path, _ := filepath.Abs(env["GITHUB_OUTPUT"])

	createOutFile(t, path)

	getenv := func(key string) string {
		return env[key]
	}
	github := newGitHubContext(getenv)

	github.SetOutput(OutputMap{
		"k": &testOutput{val: "v"},
		"v": &testOutput{val: "k"},
	})

	err := github.CloseOutput()
	if err != nil {
		t.Fatalf("error closing output: %s", err.Error())
	}
}

func Test_GitHubContext(t *testing.T) {
	env := getEnvMock(t)
	// mock getenv func
	getenv := func(key string) string {
		return env[key]
	}
	github := newGitHubContext(getenv)

	actualID := github.ID()
	expectedID := fmt.Sprintf("gha-%s-%s", env["GITHUB_RUN_ID"], env["GITHUB_RUN_NUMBER"])

	if strings.Compare(expectedID, actualID) != 0 {
		t.Errorf("expected %s, but received: %s", expectedID, actualID)
	}

	sha := env["GITHUB_SHA"]
	actualSHA := github.SHA()
	if strings.Compare(sha, actualSHA) != 0 {
		t.Errorf("expected %s, but received: %s", sha, actualSHA)
	}
}
