// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package environment

import (
	"os"
	"strings"
	"testing"
)

func TestCloseOutput(t *testing.T) {
	// Dummy env generation
	getenv := func(k string) string {
		return "something"
	}

	// Create Gitlab CI context
	gitlab := newGitLabContext(getenv)

	gitlab.SetOutput(OutputMap{
		"k1":      &testOutput{val: "v1"},
		"k2":      &testOutput{val: "v2"},
		"k3":      &testOutput{val: "v3"},
		"payload": &testOutput{val: `{"pk": "pv"}`, multiLine: true},
	})

	// Call subject
	err := gitlab.CloseOutput()

	// Assertions
	if err != nil {
		t.Fatalf("close output error: %v\n", err)
	}

	// .env is created
	contents, err := os.ReadFile(".env")
	if err != nil {
		t.Fatalf("file read error: %v\n", err)
	}

	// .env is not empty
	if len(contents) == 0 {
		t.Fatalf("no contents in output file: %v\n", err)
	}

	// Read env vars in .env into a map
	lines := strings.Split(string(contents), "\n")
	envs := make(map[string]string)
	for _, l := range lines {
		kv := strings.Split(l, "=")
		if len(kv) != 2 {
			t.Fatalf("line %s was not in correct dotenv format: %v", l, err)
		}
		envs[kv[0]] = kv[1]
	}

	// Assert env vars are same as gitlab.output
	for k, v := range gitlab.output {
		// Special key writing for keys that should be written to their own artifacts
		if v.MultiLine() {
			f := generateArtifactFileName("json", gitlab.jobName, k)
			contents, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("artifact file %s, read error: %v\n", f, err)
			}

			if len(contents) == 0 {
				t.Fatalf("no contents in artifact file %s: %v\n", f, err)
			}
			os.Remove(f)
			continue
		}

		// General keys should be in the .env file
		actual, exists := envs[k]
		if !exists {
			t.Fatalf("%s was not stored in outputfile", k)
		}

		if actual != v.Value() {
			t.Fatalf("value %s for %s expected, but found %s", v, k, actual)
		}
	}

	// Cleanup
	os.Remove(".env")

}
