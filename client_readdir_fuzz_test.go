//go:build go1.18
// +build go1.18

package smb2

import "testing"

func FuzzDecodeDirectoryEntries(f *testing.F) {
	f.Add([]byte{})
	f.Add(directoryEntryBuffer(63, 0, 0))
	f.Add(directoryEntryBuffer(64, 0, 0))
	f.Add(directoryEntryWithName(66, 0, "a"))
	f.Add(append(directoryEntryWithName(72, 72, "a"), directoryEntryWithName(66, 0, "b")...))
	f.Add(append(directoryEntryBuffer(64, 64, 0), directoryEntryBuffer(64, 0, 0)...))
	f.Add(directoryEntryBuffer(64, 64, 0))
	f.Add(directoryEntryBuffer(128, 1, 0))
	f.Add(directoryEntryBuffer(129, 65, 0))
	f.Add(directoryEntryBuffer(127, 64, 0))
	f.Add(directoryEntryBuffer(65, 0, 1))
	f.Add(directoryEntryBuffer(64, 0, ^uint32(0)))
	f.Add(directoryEntryBuffer(128, ^uint32(0), 0))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 64*1024 {
			t.Skip()
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("directory entry parser panicked for %d bytes: %v", len(data), recovered)
			}
		}()
		entries, err := decodeDirectoryEntries(data)
		if err != nil {
			if entries != nil {
				t.Fatalf("malformed input returned %d partial entries", len(entries))
			}
			if _, ok := err.(*InvalidResponseError); !ok {
				t.Fatalf("error type = %T, want *InvalidResponseError", err)
			}
			return
		}
		if len(entries) > len(data)/64+1 {
			t.Fatalf("entry count %d is not bounded by input length %d", len(entries), len(data))
		}
	})
}
