package model

import (
	"strings"
	"testing"
)

func TestSplitReplyShortStaysWhole(t *testing.T) {
	chunks := SplitReply("  hello world  ")
	if len(chunks) != 1 || chunks[0] != "hello world" {
		t.Fatalf("chunks = %#v", chunks)
	}
}

func TestSplitReplyEmpty(t *testing.T) {
	if chunks := SplitReply("   \n  "); chunks != nil {
		t.Fatalf("chunks = %#v, want nil", chunks)
	}
}

func TestSplitReplyPrefersNewline(t *testing.T) {
	first := strings.Repeat("a", MaxReplyRunes-3)
	text := first + "\n" + strings.Repeat("b", 10)
	chunks := SplitReply(text)
	if len(chunks) != 2 {
		t.Fatalf("len(chunks) = %d, want 2", len(chunks))
	}
	if chunks[0] != first {
		t.Fatalf("chunk[0] should end at newline, got trailing %q", chunks[0][len(chunks[0])-5:])
	}
	if chunks[1] != strings.Repeat("b", 10) {
		t.Fatalf("chunk[1] = %q", chunks[1])
	}
}

func TestSplitReplyRuneSafe(t *testing.T) {
	text := strings.Repeat("中", MaxReplyRunes+500)
	chunks := SplitReply(text)
	if len(chunks) != 2 {
		t.Fatalf("len(chunks) = %d, want 2", len(chunks))
	}
	for _, chunk := range chunks {
		for _, r := range chunk {
			if r != '中' {
				t.Fatalf("chunk contains broken rune %q", r)
			}
		}
	}
	if got := len([]rune(chunks[0])); got > MaxReplyRunes {
		t.Fatalf("chunk[0] rune count = %d, exceeds limit", got)
	}
}
