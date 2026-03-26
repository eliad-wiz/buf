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
	"encoding/binary"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"
)

// UnmarshalLengthDelimitedAll reads all length-delimited messages from data.
//
// Each message is encoded as a varint length prefix followed by that many bytes
// of serialized protobuf message. The newMessage function is called to create a
// new empty message instance for each message in the stream.
func UnmarshalLengthDelimitedAll(
	data []byte,
	resolver Resolver,
	newMessage func() proto.Message,
) ([]proto.Message, error) {
	if resolver == nil {
		resolver = EmptyResolver
	}
	unmarshaler := newWireUnmarshaler(resolver)
	var messages []proto.Message
	offset := 0
	for offset < len(data) {
		messageLength, bytesRead := binary.Uvarint(data[offset:])
		if bytesRead <= 0 {
			return nil, fmt.Errorf("failed to read varint at offset %d", offset)
		}
		offset += bytesRead
		if offset+int(messageLength) > len(data) {
			return nil, fmt.Errorf(
				"truncated message at offset %d: expected %d bytes but only %d remaining",
				offset, messageLength, len(data)-offset,
			)
		}
		messageData := data[offset : offset+int(messageLength)]
		offset += int(messageLength)
		message := newMessage()
		if err := unmarshaler.Unmarshal(messageData, message); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message at offset %d: %w", offset-int(messageLength), err)
		}
		messages = append(messages, message)
	}
	return messages, nil
}

// MarshalLengthDelimitedAll writes all messages in length-delimited format.
//
// Each message is encoded as a varint length prefix followed by the serialized
// protobuf message bytes.
func MarshalLengthDelimitedAll(messages []proto.Message) ([]byte, error) {
	marshaler := newWireMarshaler()
	var result []byte
	varintBuf := make([]byte, binary.MaxVarintLen64)
	for i, message := range messages {
		data, err := marshaler.Marshal(message)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal message %d: %w", i, err)
		}
		n := binary.PutUvarint(varintBuf, uint64(len(data)))
		result = append(result, varintBuf[:n]...)
		result = append(result, data...)
	}
	return result, nil
}

// WriteLengthDelimited writes a single message in length-delimited format to the writer.
func WriteLengthDelimited(w io.Writer, message proto.Message) error {
	marshaler := newWireMarshaler()
	data, err := marshaler.Marshal(message)
	if err != nil {
		return err
	}
	varintBuf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(varintBuf, uint64(len(data)))
	if _, err := w.Write(varintBuf[:n]); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
