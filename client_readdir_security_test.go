package smb2

import (
	"encoding/binary"
	"os"
	"testing"
	"unicode/utf16"
)

func directoryEntryBuffer(length int, next, fileNameLength uint32) []byte {
	data := make([]byte, length)
	if length >= 4 {
		binary.LittleEndian.PutUint32(data[0:4], next)
	}
	if length >= 64 {
		binary.LittleEndian.PutUint32(data[60:64], fileNameLength)
	}
	return data
}

func directoryEntryWithName(length int, next uint32, name string) []byte {
	encoded := utf16.Encode([]rune(name))
	fileNameLength := len(encoded) * 2
	if length < 64+fileNameLength {
		panic("test directory entry is too short for its file name")
	}

	data := directoryEntryBuffer(length, next, uint32(fileNameLength))
	for i, codeUnit := range encoded {
		binary.LittleEndian.PutUint16(data[64+i*2:], codeUnit)
	}
	return data
}

func decodeDirectoryEntriesWithoutPanic(t *testing.T, data []byte) (entries []os.FileInfo, err error) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("decodeDirectoryEntries panicked: %v", recovered)
		}
	}()

	return decodeDirectoryEntries(data)
}

func requireInvalidDirectoryResponse(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("decodeDirectoryEntries returned nil error, want invalid response")
	}
	if _, ok := err.(*InvalidResponseError); !ok {
		t.Fatalf("error type = %T, want *InvalidResponseError", err)
	}
}

func TestDecodeDirectoryEntriesRejectsShortBuffers(t *testing.T) {
	for length := 0; length < 64; length++ {
		entries, err := decodeDirectoryEntriesWithoutPanic(t, make([]byte, length))
		if entries != nil {
			t.Fatalf("buffer length %d returned %d entries, want nil", length, len(entries))
		}
		requireInvalidDirectoryResponse(t, err)
	}
}

func TestDecodeDirectoryEntriesValidOffsets(t *testing.T) {
	t.Run("terminal entry", func(t *testing.T) {
		entries, err := decodeDirectoryEntriesWithoutPanic(t, directoryEntryWithName(66, 0, "a"))
		if err != nil {
			t.Fatalf("decodeDirectoryEntries() error = %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("entry count = %d, want 1", len(entries))
		}
		if entries[0].Name() != "a" {
			t.Fatalf("entry name = %q, want %q", entries[0].Name(), "a")
		}
	})

	t.Run("two entries", func(t *testing.T) {
		data := append(directoryEntryWithName(72, 72, "a"), directoryEntryWithName(66, 0, "b")...)
		entries, err := decodeDirectoryEntriesWithoutPanic(t, data)
		if err != nil {
			t.Fatalf("decodeDirectoryEntries() error = %v", err)
		}
		if len(entries) != 2 {
			t.Fatalf("entry count = %d, want 2", len(entries))
		}
		if entries[0].Name() != "a" || entries[1].Name() != "b" {
			t.Fatalf("entry names = %q, %q; want %q, %q", entries[0].Name(), entries[1].Name(), "a", "b")
		}
	})

	t.Run("terminal entry ignores trailing padding", func(t *testing.T) {
		for padding := 0; padding < 8; padding++ {
			data := directoryEntryWithName(66+padding, 0, "z")
			for i := 66; i < len(data); i++ {
				data[i] = 0xa5
			}
			entries, err := decodeDirectoryEntriesWithoutPanic(t, data)
			if err != nil {
				t.Fatalf("padding %d: decodeDirectoryEntries() error = %v", padding, err)
			}
			if len(entries) != 1 || entries[0].Name() != "z" {
				t.Fatalf("padding %d: entries = %v, want one z entry", padding, entries)
			}
		}
	})
}

func TestDecodeDirectoryEntriesRejectsInvalidOffsets(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "offset equals buffer length", data: directoryEntryWithName(66, 66, "a")},
		{name: "offset exceeds buffer length", data: directoryEntryWithName(66, 67, "a")},
		{name: "maximum offset", data: directoryEntryWithName(128, ^uint32(0), "a")},
		{name: "one byte offset", data: directoryEntryWithName(128, 1, "a")},
		{name: "offset below fixed entry size", data: directoryEntryWithName(128, 63, "a")},
		{name: "65 byte offset", data: directoryEntryWithName(129, 65, "a")},
		{name: "misaligned offset after entry", data: directoryEntryWithName(137, 73, "a")},
		{name: "offset overlaps file name", data: directoryEntryWithName(130, 64, "a")},
		{name: "offset leaves short tail", data: directoryEntryWithName(135, 72, "a")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entries, err := decodeDirectoryEntriesWithoutPanic(t, test.data)
			if entries != nil {
				t.Fatalf("returned %d partial entries, want nil", len(entries))
			}
			requireInvalidDirectoryResponse(t, err)
		})
	}
}

func TestDecodeDirectoryEntriesRejectsOverflowingFileNameLength(t *testing.T) {
	data := directoryEntryBuffer(64, 0, ^uint32(0))
	entries, err := decodeDirectoryEntriesWithoutPanic(t, data)
	if entries != nil {
		t.Fatalf("returned %d entries, want nil", len(entries))
	}
	requireInvalidDirectoryResponse(t, err)
}

func TestDecodeDirectoryEntriesRejectsOddFileNameLength(t *testing.T) {
	data := directoryEntryBuffer(65, 0, 1)
	entries, err := decodeDirectoryEntriesWithoutPanic(t, data)
	if entries != nil {
		t.Fatalf("returned %d entries, want nil", len(entries))
	}
	requireInvalidDirectoryResponse(t, err)
}

func TestDecodeDirectoryEntriesRejectsMalformedEntryAfterValidEntry(t *testing.T) {
	data := append(directoryEntryWithName(72, 72, "a"), directoryEntryBuffer(65, 0, 2)...)
	entries, err := decodeDirectoryEntriesWithoutPanic(t, data)
	if entries != nil {
		t.Fatalf("returned %d partial entries, want nil", len(entries))
	}
	requireInvalidDirectoryResponse(t, err)
}
