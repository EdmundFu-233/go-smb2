package smb2

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func queryDirectoryResponseBuffer(length int, offset uint16, outputLength uint32) []byte {
	data := make([]byte, length)
	if length >= 2 {
		binary.LittleEndian.PutUint16(data[0:2], 9)
	}
	if length >= 4 {
		binary.LittleEndian.PutUint16(data[2:4], offset)
	}
	if length >= 8 {
		binary.LittleEndian.PutUint32(data[4:8], outputLength)
	}
	return data
}

func inspectQueryDirectoryResponse(t *testing.T, data []byte) (invalid bool, output []byte) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("QueryDirectoryResponseDecoder panicked: %v", recovered)
		}
	}()

	decoder := QueryDirectoryResponseDecoder(data)
	return decoder.IsInvalid(), decoder.OutputBuffer()
}

func TestQueryDirectoryResponseDecoderRejectsShortBuffers(t *testing.T) {
	for length := 0; length < 8; length++ {
		invalid, output := inspectQueryDirectoryResponse(t, make([]byte, length))
		if !invalid {
			t.Fatalf("buffer length %d was accepted, want invalid", length)
		}
		if output != nil {
			t.Fatalf("buffer length %d returned output, want nil", length)
		}
	}
}

func TestQueryDirectoryResponseDecoderBoundsOutput(t *testing.T) {
	validOutput := queryDirectoryResponseBuffer(72, 72, 64)
	copy(validOutput[8:], bytes.Repeat([]byte{0xa5}, 64))
	unalignedOutput := queryDirectoryResponseBuffer(73, 73, 64)
	copy(unalignedOutput[9:], bytes.Repeat([]byte{0x5a}, 64))

	tests := []struct {
		name        string
		data        []byte
		wantInvalid bool
		wantOutput  []byte
	}{
		{
			name:       "valid output",
			data:       validOutput,
			wantOutput: bytes.Repeat([]byte{0xa5}, 64),
		},
		{
			name:       "unaligned output offset allowed",
			data:       unalignedOutput,
			wantOutput: bytes.Repeat([]byte{0x5a}, 64),
		},
		{
			name:        "offset before response body",
			data:        queryDirectoryResponseBuffer(8, 71, 0),
			wantInvalid: true,
		},
		{
			name:        "output exceeds response",
			data:        queryDirectoryResponseBuffer(72, 72, 65),
			wantInvalid: true,
		},
		{
			name:        "output length overflows uint32 arithmetic",
			data:        queryDirectoryResponseBuffer(8, 72, ^uint32(0)),
			wantInvalid: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			invalid, output := inspectQueryDirectoryResponse(t, test.data)
			if invalid != test.wantInvalid {
				t.Fatalf("IsInvalid() = %v, want %v", invalid, test.wantInvalid)
			}
			if !bytes.Equal(output, test.wantOutput) {
				t.Fatalf("output = %x, want %x", output, test.wantOutput)
			}
			if test.wantInvalid && output != nil {
				t.Fatal("invalid response returned a non-nil output buffer")
			}
		})
	}
}
