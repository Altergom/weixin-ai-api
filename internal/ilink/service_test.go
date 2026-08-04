package ilink

import "testing"

func TestNormalizeStatus(t *testing.T) {
	if normalizeStatus("scanned") != "scaned" {
		t.Fatal("scanned should match iLink compatibility spelling")
	}
}
