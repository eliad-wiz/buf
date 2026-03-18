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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestConvertBytesValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "valid_utf8",
			input:    []byte("hello world"),
			expected: "hello world",
		},
		{
			name:     "invalid_utf8",
			input:    []byte{0xff, 0xfe, 0xfd},
			expected: "base64:" + base64.StdEncoding.EncodeToString([]byte{0xff, 0xfe, 0xfd}),
		},
		{
			name:     "empty",
			input:    []byte{},
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			encoded := base64.StdEncoding.EncodeToString(tt.input)
			result := convertBytesValue(encoded)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// buildTestMessageDescriptor builds a FileDescriptorProto with a message that has a bytes field
// and returns the message descriptor.
func buildTestMessageDescriptor(t *testing.T) protoreflect.MessageDescriptor {
	t.Helper()
	bytesLabel := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	bytesType := descriptorpb.FieldDescriptorProto_TYPE_BYTES
	stringType := descriptorpb.FieldDescriptorProto_TYPE_STRING
	messageType := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	repeatedLabel := descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	bytesFieldName := "data"
	stringFieldName := "name"
	nestedFieldName := "nested"
	repeatedBytesFieldName := "items"
	mapFieldName := "tags"
	var bytesFieldNum int32 = 1
	var stringFieldNum int32 = 2
	var nestedFieldNum int32 = 3
	var repeatedBytesFieldNum int32 = 4
	var mapFieldNum int32 = 5
	mapEntryName := "TagsEntry"
	mapKeyName := "key"
	mapValueName := "value"
	var mapKeyNum int32 = 1
	var mapValueNum int32 = 2
	isMapEntry := true
	nestedMsgName := ".test.TestMessage"
	mapEntryFullName := ".test.TestMessage.TagsEntry"
	syntax := "proto3"

	fdp := &descriptorpb.FileDescriptorProto{
		Name:   proto.String("test.proto"),
		Syntax: &syntax,
		Package: proto.String("test"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("TestMessage"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   &bytesFieldName,
						Number: &bytesFieldNum,
						Label:  &bytesLabel,
						Type:   &bytesType,
					},
					{
						Name:   &stringFieldName,
						Number: &stringFieldNum,
						Label:  &bytesLabel,
						Type:   &stringType,
					},
					{
						Name:     &nestedFieldName,
						Number:   &nestedFieldNum,
						Label:    &bytesLabel,
						Type:     &messageType,
						TypeName: &nestedMsgName,
					},
					{
						Name:   &repeatedBytesFieldName,
						Number: &repeatedBytesFieldNum,
						Label:  &repeatedLabel,
						Type:   &bytesType,
					},
					{
						Name:     &mapFieldName,
						Number:   &mapFieldNum,
						Label:    &repeatedLabel,
						Type:     &messageType,
						TypeName: &mapEntryFullName,
					},
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: &mapEntryName,
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   &mapKeyName,
								Number: &mapKeyNum,
								Label:  &bytesLabel,
								Type:   &stringType,
							},
							{
								Name:   &mapValueName,
								Number: &mapValueNum,
								Label:  &bytesLabel,
								Type:   &bytesType,
							},
						},
						Options: &descriptorpb.MessageOptions{
							MapEntry: &isMapEntry,
						},
					},
				},
			},
		},
	}

	fd, err := protodesc.NewFile(fdp, nil)
	require.NoError(t, err)
	return fd.Messages().Get(0)
}

func TestTransformBytesAsText_SimpleBytes(t *testing.T) {
	t.Parallel()
	msgDesc := buildTestMessageDescriptor(t)
	msg := dynamicpb.NewMessage(msgDesc)

	msg.Set(msgDesc.Fields().ByName("data"), protoreflect.ValueOfBytes([]byte("hello")))
	msg.Set(msgDesc.Fields().ByName("name"), protoreflect.ValueOfString("test"))

	resolver, err := NewResolver(mustFileDescriptorProto(t, msgDesc))
	require.NoError(t, err)

	marshaler := NewJSONMarshaler(resolver, JSONMarshalerWithBytesAsText())
	data, err := marshaler.Marshal(msg)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"hello"`)
	assert.NotContains(t, string(data), base64.StdEncoding.EncodeToString([]byte("hello")))
}

func TestTransformBytesAsText_InvalidUTF8(t *testing.T) {
	t.Parallel()
	msgDesc := buildTestMessageDescriptor(t)
	msg := dynamicpb.NewMessage(msgDesc)

	invalidBytes := []byte{0xff, 0xfe}
	msg.Set(msgDesc.Fields().ByName("data"), protoreflect.ValueOfBytes(invalidBytes))

	resolver, err := NewResolver(mustFileDescriptorProto(t, msgDesc))
	require.NoError(t, err)

	marshaler := NewJSONMarshaler(resolver, JSONMarshalerWithBytesAsText())
	data, err := marshaler.Marshal(msg)
	require.NoError(t, err)
	expected := "base64:" + base64.StdEncoding.EncodeToString(invalidBytes)
	assert.Contains(t, string(data), expected)
}

func TestTransformBytesAsText_NestedMessage(t *testing.T) {
	t.Parallel()
	msgDesc := buildTestMessageDescriptor(t)
	msg := dynamicpb.NewMessage(msgDesc)
	nested := dynamicpb.NewMessage(msgDesc)
	nested.Set(msgDesc.Fields().ByName("data"), protoreflect.ValueOfBytes([]byte("nested-value")))
	msg.Set(msgDesc.Fields().ByName("nested"), protoreflect.ValueOfMessage(nested))

	resolver, err := NewResolver(mustFileDescriptorProto(t, msgDesc))
	require.NoError(t, err)

	marshaler := NewJSONMarshaler(resolver, JSONMarshalerWithBytesAsText())
	data, err := marshaler.Marshal(msg)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"nested-value"`)
}

func TestTransformBytesAsText_RepeatedBytes(t *testing.T) {
	t.Parallel()
	msgDesc := buildTestMessageDescriptor(t)
	msg := dynamicpb.NewMessage(msgDesc)

	itemsField := msgDesc.Fields().ByName("items")
	list := msg.NewField(itemsField).List()
	list.Append(protoreflect.ValueOfBytes([]byte("valid-utf8")))
	list.Append(protoreflect.ValueOfBytes([]byte{0x80, 0x81}))
	msg.Set(itemsField, protoreflect.ValueOfList(list))

	resolver, err := NewResolver(mustFileDescriptorProto(t, msgDesc))
	require.NoError(t, err)

	marshaler := NewJSONMarshaler(resolver, JSONMarshalerWithBytesAsText())
	data, err := marshaler.Marshal(msg)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"valid-utf8"`)
	assert.Contains(t, string(data), "base64:")
}

func TestTransformBytesAsText_MapBytesValue(t *testing.T) {
	t.Parallel()
	msgDesc := buildTestMessageDescriptor(t)
	msg := dynamicpb.NewMessage(msgDesc)

	tagsField := msgDesc.Fields().ByName("tags")
	mapField := msg.NewField(tagsField).Map()
	mapField.Set(
		protoreflect.ValueOfString("key1").MapKey(),
		protoreflect.ValueOfBytes([]byte("map-value")),
	)
	msg.Set(tagsField, protoreflect.ValueOfMap(mapField))

	resolver, err := NewResolver(mustFileDescriptorProto(t, msgDesc))
	require.NoError(t, err)

	marshaler := NewJSONMarshaler(resolver, JSONMarshalerWithBytesAsText())
	data, err := marshaler.Marshal(msg)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"map-value"`)
}

func TestTransformBytesAsText_UseProtoNames(t *testing.T) {
	t.Parallel()
	msgDesc := buildTestMessageDescriptor(t)
	msg := dynamicpb.NewMessage(msgDesc)
	msg.Set(msgDesc.Fields().ByName("data"), protoreflect.ValueOfBytes([]byte("proto-names")))

	resolver, err := NewResolver(mustFileDescriptorProto(t, msgDesc))
	require.NoError(t, err)

	marshaler := NewJSONMarshaler(resolver, JSONMarshalerWithBytesAsText(), JSONMarshalerWithUseProtoNames())
	data, err := marshaler.Marshal(msg)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"proto-names"`)
	assert.Contains(t, string(data), `"data"`)
}

func TestTransformBytesAsText_EmptyBytes(t *testing.T) {
	t.Parallel()
	msgDesc := buildTestMessageDescriptor(t)
	msg := dynamicpb.NewMessage(msgDesc)
	// Empty bytes fields are not emitted by default in proto3.
	// With emitUnpopulated, they'd show as "".
	// Just verify no error.
	resolver, err := NewResolver(mustFileDescriptorProto(t, msgDesc))
	require.NoError(t, err)

	marshaler := NewJSONMarshaler(resolver, JSONMarshalerWithBytesAsText(), JSONMarshalerWithEmitUnpopulated())
	data, err := marshaler.Marshal(msg)
	require.NoError(t, err)
	// The empty bytes field should appear as empty string.
	assert.Contains(t, string(data), `"data":""`)
}

func mustFileDescriptorProto(t *testing.T, msgDesc protoreflect.MessageDescriptor) *descriptorpb.FileDescriptorProto {
	t.Helper()
	return protodesc.ToFileDescriptorProto(msgDesc.ParentFile())
}
