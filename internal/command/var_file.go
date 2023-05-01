// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"log"
	"os"
	"strings"

	"github.com/hashicorp/go-tfe"
)

const VarEnvPrefix = "TF_VAR_"

func collectVariables() []*tfe.RunVariable {
	var tfVars []*tfe.RunVariable
	// get vars from env
	tfVarMap := collectEnvVariables()
	for _, value := range tfVarMap {
		tfVars = append(tfVars, value)
	}
	return tfVars
}

func collectEnvVariables() map[string]*tfe.RunVariable {
	tfRunMap := make(map[string]*tfe.RunVariable)

	env := os.Environ()
	for _, v := range env {
		if !strings.HasPrefix(v, VarEnvPrefix) {
			continue
		}
		eq := strings.Index(v, "=")
		if eq == -1 {
			continue
		}

		key := v[len(VarEnvPrefix):eq]
		value := v[eq+1:]

		log.Printf("[DEBUG] adding variable: '%s', with: '%s'", key, value)

		tfRunMap[key] = &tfe.RunVariable{
			Key:   key,
			Value: value,
		}
	}
	return tfRunMap
}
