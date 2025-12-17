// Copyright IBM Corp. 2023, 2025
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/jsonapi"
)

type outputMessage struct {
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

func (o *outputMessage) IncludeWithPlatform() bool {
	return o.platformOut
}

func (o *outputMessage) Value() (string, error) {
	switch o.value.(type) {
	case string:
		return o.value.(string), nil
	default:
		reflectVal := reflect.ValueOf(o.value)
		reflectInd := reflect.Indirect(reflectVal)
		refType := reflectInd.Type()
		// if type is not a struct, return as marshaled string
		if refType.Kind() != reflect.Struct {
			b, bErr := json.Marshal(o.value)
			if bErr != nil {
				return "", bErr
			}
			return string(b), bErr
		}

		// github.com/hashicorp/go-tfe structs use `jsonapi` tag annotation and requires marshalling
		// with the github.com/hashicorp/jsonapi package to serialize to the correct format.
		// depending on the structs tags, we will use `jsonapi` or standard `json` marhshaler.
		switch resolveMarshaler(refType) {
		// we detected jsonapi, use jsonapi.MarshalPayload. eg. *tfe.Run struct
		case JSONAPI:
			return marshalJsonAPI(o.value)
		// detected json tag annotations
		case JSON:
			return marshalJson(o.value)
		default:
			return "", fmt.Errorf("no marshaller found for %v", refType)
		}
	}
}

func (o *outputMessage) MultiLine() bool {
	return o.multiLine
}

var defaultOutputOpts = &outputOpts{
	stdOut:      true,
	platformOut: true,
	multiLine:   false,
}

type outputOpts struct {
	// option that indicates if value should be displayed to stdout
	stdOut bool
	// option to include value to platform output when detected
	platformOut bool
	// option to indicate if value contains a multiline value as some platforms: gitlab do not support multiline values in `.env`
	multiLine bool
}

func newOutputMessage(name string, value interface{}, opts *outputOpts) *outputMessage {
	return &outputMessage{
		name:        name,
		value:       value,
		stdOut:      opts.stdOut,
		platformOut: opts.platformOut,
		multiLine:   opts.multiLine,
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
