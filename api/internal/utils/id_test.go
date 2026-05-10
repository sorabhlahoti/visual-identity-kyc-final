package utils

import (
	"regexp"
	"strings"
	"testing"
)

func TestNewIDKeepsPrefixForTransactions(t *testing.T) {
	got := NewID("txn")
	if !strings.HasPrefix(got, "txn_") {
		t.Fatalf("expected txn_ prefix, got %q", got)
	}
}

func TestNewUUIDIsQdrantCompatible(t *testing.T) {
	got := NewUUID()
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !re.MatchString(got) {
		t.Fatalf("not a valid UUID v4: %q", got)
	}
}
