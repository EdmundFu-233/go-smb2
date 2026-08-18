package smb2

import (
	"testing"

	. "github.com/EdmundFu-233/go-smb2/internal/smb2"
)

func TestValidNegotiatedPayloadSizes(t *testing.T) {
	minimum := uint32(singleCreditMaxPayloadSize)
	tests := []struct {
		name                  string
		transact, read, write uint32
		want                  bool
	}{
		{name: "minimum", transact: minimum, read: minimum, write: minimum, want: true},
		{name: "maximum", transact: ^uint32(0), read: ^uint32(0), write: ^uint32(0), want: true},
		{name: "zero transact", transact: 0, read: minimum, write: minimum},
		{name: "short transact", transact: minimum - 1, read: minimum, write: minimum},
		{name: "short read", transact: minimum, read: minimum - 1, write: minimum},
		{name: "short write", transact: minimum, read: minimum, write: minimum - 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validNegotiatedPayloadSizes(test.transact, test.read, test.write); got != test.want {
				t.Fatalf("validNegotiatedPayloadSizes() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestEffectiveMaxPayloadSizeBoundsBeforeIntConversion(t *testing.T) {
	tests := []struct {
		name         string
		size         uint32
		capabilities uint32
		want         int
	}{
		{name: "minimum single credit", size: 64 * 1024, want: 64 * 1024},
		{name: "maximum single credit", size: ^uint32(0), want: 64 * 1024},
		{name: "minimum large MTU", size: 64 * 1024, capabilities: SMB2_GLOBAL_CAP_LARGE_MTU, want: 64 * 1024},
		{name: "one megabyte large MTU", size: 1024 * 1024, capabilities: SMB2_GLOBAL_CAP_LARGE_MTU, want: 1024 * 1024},
		{name: "above one megabyte large MTU", size: 1024*1024 + 1, capabilities: SMB2_GLOBAL_CAP_LARGE_MTU, want: 1024 * 1024},
		{name: "max int32 large MTU", size: 1<<31 - 1, capabilities: SMB2_GLOBAL_CAP_LARGE_MTU, want: 1024 * 1024},
		{name: "int32 overflow boundary large MTU", size: 1 << 31, capabilities: SMB2_GLOBAL_CAP_LARGE_MTU, want: 1024 * 1024},
		{name: "maximum uint32 large MTU", size: ^uint32(0), capabilities: SMB2_GLOBAL_CAP_LARGE_MTU, want: 1024 * 1024},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := effectiveMaxPayloadSize(test.size, test.capabilities); got != test.want {
				t.Fatalf("effectiveMaxPayloadSize() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestFilePayloadSizeHelpersBoundHighAdvertisements(t *testing.T) {
	connection := &conn{
		capabilities:    SMB2_GLOBAL_CAP_LARGE_MTU,
		maxTransactSize: ^uint32(0),
		maxReadSize:     ^uint32(0),
		maxWriteSize:    ^uint32(0),
	}
	file := &File{fs: &Share{treeConn: &treeConn{session: &session{conn: connection}}}}

	if got := file.maxTransactSize(); got != winMaxPayloadSize {
		t.Fatalf("maxTransactSize() = %d, want %d", got, winMaxPayloadSize)
	}
	if got := file.maxReadSize(); got != winMaxPayloadSize {
		t.Fatalf("maxReadSize() = %d, want %d", got, winMaxPayloadSize)
	}
	if got := file.maxWriteSize(); got != winMaxPayloadSize {
		t.Fatalf("maxWriteSize() = %d, want %d", got, winMaxPayloadSize)
	}
}
