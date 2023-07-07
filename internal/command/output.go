// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/jsonapi"
)

type OutputMessage struct {
	// unique name
	name string
	// raw value to return to stdout, platform
	value interface{}
	// determines if message should be included in stdout. default: true
	stdOut bool
	// determines if message should be sent to platform. default: true
	platformOut bool
	// if the value may contain strings/json that is multiline
	multiLine bool
}

func (o *OutputMessage) IncludeWithPlatform() bool {
	return o.platformOut
}

func (o *OutputMessage) String() (sValue string) {
	switch o.value.(type) {
	case string:
		sValue = fmt.Sprintf("%s", o.value)
	default:
		reflectVal := reflect.ValueOf(o.value)
		reflectInd := reflect.Indirect(reflectVal)
		refType := reflectInd.Type()
		// if type is not a struct, return as marshaled string
		if refType.Kind() != reflect.Struct {
			b, _ := json.Marshal(o.value)
			sValue = string(b)
			return
		}

		// depending on presence of jsonapi struct tags, determine marshaller
		switch resolveMarshaler(refType) {
		case JSONAPI:
			// we detected jsonapi, use jsonapi.MarshalPayload. eg. *tfe.Run struct
			sValue, _ = marshalJsonAPI(o.value)
		default:
			// everything else use standard json.Marshal
			sValue, _ = marshalJson(o.value)
		}
	}
	return
}

func (o *OutputMessage) MultiLine() bool {
	return o.multiLine
}

func newOutputMessage(name string, value interface{}) *OutputMessage {
	return &OutputMessage{
		name:        name,
		value:       value,
		stdOut:      true,
		platformOut: true,
		multiLine:   false,
	}
}

type Marshaler string

const JSONAPI Marshaler = "jsonapi"
const JSON Marshaler = "json"

// use reflection type to check for struct field tags: `json` | `jsonapi`
func resolveMarshaler(t reflect.Type) Marshaler {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get(string(JSON))
		apiTag := field.Tag.Get(string(JSONAPI))
		if apiTag != "" {
			return JSONAPI
		}
		if jsonTag != "" {
			return JSON
		}
	}
	return ""
}

func marshalJson(data interface{}) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// marshal structs decorated with `jsonapi` fields
func marshalJsonAPI(data interface{}) (string, error) {
	buffer := new(bytes.Buffer)

	err := jsonapi.MarshalPayload(buffer, data)
	if err != nil {
		return "", err
	}

	outJson := buffer.String()

	return outJson, nil
}
