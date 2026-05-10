package security

import "testing"

func TestDemographicHashDeterministicAndPeppered(t *testing.T) {
	a := DemographicHash("pepper-a", "1990-01-01", "M")
	b := DemographicHash("pepper-a", " 1990-01-01 ", "m")
	c := DemographicHash("pepper-b", "1990-01-01", "M")
	if a != b {
		t.Fatalf("expected normalized deterministic hash")
	}
	if a == c {
		t.Fatalf("expected pepper to change hash")
	}
}
