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
	//
	value interface{}
	// determines if message should be included in stdout. default: true
	stdOut bool
	// determines if message should be sent to platform. default: true
	platformOut bool
	// if the value may contain strings/json that is multiline
	multiLine bool
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
		// if type is not a struct, return
		if refType.Kind() != reflect.Struct {
			b, _ := json.Marshal(o.value)
			sValue = string(b)
			return
		}
		// check struct tags for `json` or `jsonapi`
		json, jsonAPI := structHasJsonTags(refType)
		// if json, normal marshal
		if json {
			sValue, _ = marshalJson(o.value)
			return
		}
		// use jsonapi marshal
		if jsonAPI {
			sValue, _ = marshalJsonAPI(o.value)
			return
		}
	}
	return
}

func structHasJsonTags(t reflect.Type) (json bool, jsonAPI bool) {
	jsonLabel, apiLabel := "json", "jsonapi"
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Get the field tag value
		jsonTag := field.Tag.Get(jsonLabel)
		apiTag := field.Tag.Get(apiLabel)
		// as soon as we determine, return
		if jsonTag != "" {
			fmt.Println("found json tag")
			json = true
			return
		}
		if apiTag != "" {
			jsonAPI = true
			return
		}
	}
	return json, jsonAPI
}

func marshalJson(data interface{}) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func marshalJsonAPI(data interface{}) (string, error) {
	buffer := new(bytes.Buffer)

	err := jsonapi.MarshalPayload(buffer, data)
	if err != nil {
		return "", err
	}

	outJson := buffer.String()

	return outJson, nil
}
