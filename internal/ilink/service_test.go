package ilink

import "testing"

func TestMaskKey(t *testing.T) {
	if got := maskKey("1234567890"); got != "1234***7890" {
		t.Fatalf("got %q", got)
	}
}
func TestNormalizeStatus(t *testing.T) {
	if normalizeStatus("scanned") != "scaned" {
		t.Fatal("scanned should match iLink compatibility spelling")
	}
}
