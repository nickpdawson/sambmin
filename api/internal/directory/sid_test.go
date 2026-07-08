package directory

import (
	"encoding/binary"
	"testing"
)

func TestSidToString(t *testing.T) {
	// Build the binary form of S-1-5-21-2987504718-2361157560-2967585114-1112.
	subs := []uint32{21, 2987504718, 2361157560, 2967585114, 1112}
	b := []byte{0x01, byte(len(subs)), 0, 0, 0, 0, 0, 5} // rev, count, authority=5 (big-endian)
	for _, s := range subs {
		var tmp [4]byte
		binary.LittleEndian.PutUint32(tmp[:], s)
		b = append(b, tmp[:]...)
	}

	got := sidToString(b)
	want := "S-1-5-21-2987504718-2361157560-2967585114-1112"
	if got != want {
		t.Errorf("sidToString = %q, want %q", got, want)
	}
}

func TestSidToString_Malformed(t *testing.T) {
	if sidToString(nil) != "" {
		t.Error("nil should decode to empty string")
	}
	if sidToString([]byte{0x01, 0x05, 0, 0, 0, 0, 0, 5, 0x15}) != "" {
		t.Error("truncated sub-authorities should decode to empty string")
	}
}

func TestTrusteeName(t *testing.T) {
	sidMap := map[string]Trustee{
		"S-1-5-21-1-2-3-1104": {SID: "S-1-5-21-1-2-3-1104", SamAccountName: "svc-backup"},
	}
	if got := TrusteeName("S-1-5-21-1-2-3-1104", sidMap); got != "svc-backup" {
		t.Errorf("resolved principal = %q, want svc-backup", got)
	}
	if got := TrusteeName("DA", sidMap); got != "Domain Admins" {
		t.Errorf("DA alias = %q, want Domain Admins", got)
	}
	if got := TrusteeName("S-1-5-21-9-9-9-512", sidMap); got != "Domain Admins" {
		t.Errorf("RID-512 = %q, want Domain Admins", got)
	}
	if got := TrusteeName("S-1-1-0", sidMap); got != "Everyone" {
		t.Errorf("S-1-1-0 = %q, want Everyone", got)
	}
	// Unknown token passes through unchanged.
	if got := TrusteeName("S-1-5-21-1-2-3-9999", sidMap); got != "S-1-5-21-1-2-3-9999" {
		t.Errorf("unknown = %q, want passthrough", got)
	}
}
