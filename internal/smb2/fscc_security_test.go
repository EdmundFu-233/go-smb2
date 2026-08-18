package smb2

import (
	"encoding/binary"
	"testing"
)

func fileDirectoryDecoderIsInvalid(t *testing.T, data []byte) (invalid bool) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("FileDirectoryInformationDecoder.IsInvalid panicked: %v", recovered)
		}
	}()
	return FileDirectoryInformationDecoder(data).IsInvalid()
}

func TestFileDirectoryInformationDecoderRejectsShortBuffers(t *testing.T) {
	for length := 0; length < 64; length++ {
		if !fileDirectoryDecoderIsInvalid(t, make([]byte, length)) {
			t.Fatalf("buffer length %d was accepted, want invalid", length)
		}
	}
}

func TestFileDirectoryInformationDecoderBoundsFileName(t *testing.T) {
	tests := []struct {
		name           string
		bufferLength   int
		fileNameLength uint32
		wantInvalid    bool
	}{
		{name: "empty name", bufferLength: 64, wantInvalid: true},
		{name: "exact name", bufferLength: 66, fileNameLength: 2, wantInvalid: false},
		{name: "odd UTF-16 name", bufferLength: 65, fileNameLength: 1, wantInvalid: true},
		{name: "truncated name", bufferLength: 65, fileNameLength: 2, wantInvalid: true},
		{name: "overflowing name", bufferLength: 64, fileNameLength: ^uint32(0), wantInvalid: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := make([]byte, test.bufferLength)
			binary.LittleEndian.PutUint32(data[60:64], test.fileNameLength)
			if got := fileDirectoryDecoderIsInvalid(t, data); got != test.wantInvalid {
				t.Fatalf("IsInvalid() = %v, want %v", got, test.wantInvalid)
			}
		})
	}
}
