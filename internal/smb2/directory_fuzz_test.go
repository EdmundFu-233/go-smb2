//go:build go1.18
// +build go1.18

package smb2

import (
	"encoding/binary"
	"testing"
)

func FuzzFileDirectoryInformationDecoder(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 63))
	f.Add(make([]byte, 64))

	overflow := make([]byte, 64)
	binary.LittleEndian.PutUint32(overflow[60:64], ^uint32(0))
	f.Add(overflow)

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("directory decoder panicked for %d bytes: %v", len(data), recovered)
			}
		}()

		decoder := FileDirectoryInformationDecoder(data)
		if !decoder.IsInvalid() {
			_ = decoder.FileName()
		}
	})
}

func FuzzQueryDirectoryResponseDecoder(f *testing.F) {
	f.Add([]byte{})
	f.Add(queryDirectoryResponseBuffer(8, 72, 0))
	f.Add(queryDirectoryResponseBuffer(72, 72, 64))
	f.Add(queryDirectoryResponseBuffer(8, 72, ^uint32(0)))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("query-directory response decoder panicked for %d bytes: %v", len(data), recovered)
			}
		}()

		decoder := QueryDirectoryResponseDecoder(data)
		invalid := decoder.IsInvalid()
		output := decoder.OutputBuffer()
		if invalid && output != nil {
			t.Fatal("invalid query-directory response returned a non-nil output buffer")
		}
		if len(output) > len(data) {
			t.Fatalf("output length %d exceeds input length %d", len(output), len(data))
		}
	})
}
