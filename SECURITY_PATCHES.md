# Security patch provenance

This repository is a ReCasaOS-maintained fork of
`github.com/hirochachacha/go-smb2`. Its source baseline is the upstream
`v1.1.0` tag at commit
`82da9adcf15307deb147fbe4a0732d0e0b657e2c`.

The original BSD 2-Clause `LICENSE` is retained unchanged. Source and binary
redistributions must continue to satisfy that license's notice and disclaimer
requirements.

## GO-2026-5051

The Go vulnerability report describes process panics while parsing
server-controlled SMB `QUERY_DIRECTORY` responses:

- <https://pkg.go.dev/vuln/GO-2026-5051>
- <https://github.com/golang/vulndb/issues/5051>

The reviewed upstream fix reference is CloudSoda commit
`7b96c35f5f4babfc9d68a60baf5e85f2303d4a4b`. Its later fail-closed offset
handling was merged to CloudSoda's mainline as commit
`d8c5600d73b8dc5d00f4f43a40ac0d91b8652b5e`.

This fork does not cherry-pick either mixed commit wholesale. It independently
applies the relevant bounds checks to the v1.1.0 baseline and records the
following intentional differences:

- a directory entry shorter than its 64-byte fixed header is rejected before
  any length field is read;
- file-name length arithmetic is performed in `uint64` before any conversion
  to a slice index;
- the query-directory response envelope validates offset and length in
  `uint64`, and `OutputBuffer` returns `nil` for every invalid envelope;
- only `NextEntryOffset == 0` is a successful end marker;
- a non-zero offset that overlaps the current entry, reaches or exceeds the
  remaining buffer, is not 8-byte aligned, or leaves a tail shorter than one
  fixed header returns an `InvalidResponseError` and no partial listing;
- empty or odd file-name byte lengths are rejected because a protocol filename
  contains at least one UTF-16LE code unit;
- a terminal entry is allowed to contain trailing bytes for compatibility; the
  specification does not require padding after the last entry;
- `OutputBufferOffset` is bounds-checked but is not required to be 8-byte
  aligned because the SMB2 QUERY_DIRECTORY response specification does not
  impose that requirement on this field;
- CloudSoda's unrelated decoder changes, receiver recovery, formatting
  changes, Kerberos support, and SDDL APIs are not imported by this patch.

The protocol validation is based on the following Microsoft Open
Specifications:

- [MS-FSCC section 2.4.10](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-fscc/b38bf518-9057-4c88-9ddd-5e2d3976a64b)
  requires multiple `FILE_DIRECTORY_INFORMATION` entries to be aligned on
  8-byte boundaries and requires receivers to ignore inserted alignment
  bytes;
- [MS-DTYP glossary](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-dtyp/a66edeb1-52a0-4d64-a93b-2f5c833d7d92)
  defines protocol Unicode strings as UTF-16LE code units;
- [MS-FSCC section 2.1.5.2](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-fscc/2917da5c-253c-4c0e-aaf6-9dddc37d2e6e)
  requires a filename to contain at least one character;
- [MS-SMB2 section 2.2.34](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-smb2/4f75351b-048c-4a0c-9ea3-addd55a71956)
  defines `OutputBufferOffset` as a byte offset without imposing an 8-byte
  alignment requirement on that response field.

Regression tests cover short fixed headers, overflowing file-name and response
lengths, terminal and multi-entry responses, equal and oversized offsets,
small and misaligned offsets, short tails, odd UTF-16 byte lengths, typed
errors, and no-panic fuzz seeds.

## Dependency boundary

This fork intentionally adds no dependency for the GO-2026-5051 fix. In
particular, it does not depend on either `github.com/cloudsoda/go-smb2` or
`github.com/cloudsoda/sddl`. Its existing `golang.org/x/crypto` dependency is
updated to v0.55.0; this code imports only `x/crypto/md4` for NTLM protocol
compatibility, and CI rejects any OpenPGP package in the selected dependency
graphs.

## Additional ReCasaOS hardening

The SMB2 negotiate response also supplies three server-controlled `uint32`
maximum payload sizes. Converting values at or above `2^31` to `int` before
bounding them can produce negative sizes on 32-bit targets. This fork caps the
unsigned values before conversion and rejects a negotiation if any advertised
maximum is below 64 KiB, following the MS-SMB2 client recommendation to
disconnect for such values. The 386 and armv7 test binaries exercise these
paths at runtime in CI.

The Go vulnerability database matches module paths. Because this fork uses a
new path, a clean `govulncheck` result alone is not evidence that the parser is
fixed; the regression tests and recorded source provenance remain mandatory.
