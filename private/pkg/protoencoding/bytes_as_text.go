// Copyright 2020-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package protoencoding

import (
	"encoding/base64"
	"encoding/json"
	"unicode/utf8"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// transformBytesAsText post-processes JSON-marshaled protobuf data to convert
// base64-encoded bytes fields to UTF-8 text strings. If a bytes value is not
// valid UTF-8, it is rendered as "base64:<base64_value>".
func transformBytesAsText(
	data []byte,
	messageDescriptor protoreflect.MessageDescriptor,
	useProtoNames bool,
) ([]byte, error) {
	var jsonObj map[string]any
	if err := json.Unmarshal(data, &jsonObj); err != nil {
		return nil, err
	}
	transformMessageValue(jsonObj, messageDescriptor, useProtoNames)
	return json.Marshal(jsonObj)
}

func transformMessageValue(
	obj map[string]any,
	messageDescriptor protoreflect.MessageDescriptor,
	useProtoNames bool,
) {
	fields := messageDescriptor.Fields()
	for i := 0; i < fields.Len(); i++ {
		fieldDescriptor := fields.Get(i)
		fieldName := fieldJSONName(fieldDescriptor, useProtoNames)
		value, ok := obj[fieldName]
		if !ok {
			continue
		}
		if fieldDescriptor.IsMap() {
			mapValue := fieldDescriptor.MapValue()
			if mapObj, ok := value.(map[string]any); ok {
				if mapValue.Kind() == protoreflect.BytesKind {
					for k, v := range mapObj {
						if s, ok := v.(string); ok {
							mapObj[k] = convertBytesValue(s)
						}
					}
				} else if mapValue.Kind() == protoreflect.MessageKind || mapValue.Kind() == protoreflect.GroupKind {
					for _, v := range mapObj {
						if nested, ok := v.(map[string]any); ok {
							transformMessageValue(nested, mapValue.Message(), useProtoNames)
						}
					}
				}
			}
			continue
		}
		if fieldDescriptor.IsList() {
			if arr, ok := value.([]any); ok {
				if fieldDescriptor.Kind() == protoreflect.BytesKind {
					for i, elem := range arr {
						if s, ok := elem.(string); ok {
							arr[i] = convertBytesValue(s)
						}
					}
				} else if fieldDescriptor.Kind() == protoreflect.MessageKind || fieldDescriptor.Kind() == protoreflect.GroupKind {
					for _, elem := range arr {
						if nested, ok := elem.(map[string]any); ok {
							transformMessageValue(nested, fieldDescriptor.Message(), useProtoNames)
						}
					}
				}
			}
			continue
		}
		switch fieldDescriptor.Kind() {
		case protoreflect.BytesKind:
			if s, ok := value.(string); ok {
				obj[fieldName] = convertBytesValue(s)
			}
		case protoreflect.MessageKind, protoreflect.GroupKind:
			if nested, ok := value.(map[string]any); ok {
				transformMessageValue(nested, fieldDescriptor.Message(), useProtoNames)
			}
		}
	}
}

func fieldJSONName(fieldDescriptor protoreflect.FieldDescriptor, useProtoNames bool) string {
	if useProtoNames {
		return string(fieldDescriptor.Name())
	}
	return fieldDescriptor.JSONName()
}

func convertBytesValue(base64Value string) string {
	decoded, err := base64.StdEncoding.DecodeString(base64Value)
	if err != nil {
		// Try URL-safe encoding as protojson may use either.
		decoded, err = base64.RawStdEncoding.DecodeString(base64Value)
		if err != nil {
			return "base64:" + base64Value
		}
	}
	if utf8.Valid(decoded) {
		return string(decoded)
	}
	return "base64:" + base64Value
}
