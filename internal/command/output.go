// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"bytes"

	"github.com/hashicorp/jsonapi"
)

func outputJson(data interface{}) (string, error) {
	buffer := new(bytes.Buffer)

	err := jsonapi.MarshalPayload(buffer, data)
	if err != nil {
		return "", err
	}

	outJson := buffer.String()

	return outJson, nil
}
