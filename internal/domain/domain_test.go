package domain

import (
	"regexp"
	"testing"
)

func TestRootPinTableLength(t *testing.T) {
	if got := len(RootPinTable); got != 30 {
		t.Fatalf("RootPinTable length = %d, want 30", got)
	}
}

func TestRootPinTableNamesUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range RootPinTable {
		if seen[e.Name] {
			t.Fatalf("duplicate RootPinTable name: %s", e.Name)
		}
		seen[e.Name] = true
	}
}

func TestRootPinTableSubsUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, e := range RootPinTable {
		if seen[e.Sub] {
			t.Fatalf("duplicate RootPinTable sub: %s", e.Sub)
		}
		seen[e.Sub] = true
	}
}

func TestScalarSlotsAreRootPins(t *testing.T) {
	names := map[string]bool{}
	for _, e := range RootPinTable {
		names[e.Name] = true
	}
	for slot := range ScalarSlots {
		if !names[slot] {
			t.Errorf("ScalarSlots contains %q which is not in RootPinTable", slot)
		}
	}
}

func TestDefaultRouting(t *testing.T) {
	tests := []struct {
		ot   OutputType
		want []string
	}{
		{CMOTFloat1, []string{"Emissive Color"}},
		{CMOTFloat2, []string{"Emissive Color"}},
		{CMOTFloat3, []string{"Base Color"}},
		{CMOTFloat4, []string{"Base Color", "Opacity"}},
	}
	for _, tt := range tests {
		got := DefaultRouting(tt.ot)
		if len(got) != len(tt.want) {
			t.Errorf("DefaultRouting(%s) = %v, want %v", tt.ot, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("DefaultRouting(%s)[%d] = %q, want %q", tt.ot, i, got[i], tt.want[i])
			}
		}
	}
}

var guidPattern = regexp.MustCompile(`^[0-9A-F]{32}$`)

func TestNewGUID(t *testing.T) {
	g := NewGUID()
	if !guidPattern.MatchString(g) {
		t.Fatalf("NewGUID() = %q, doesn't match [0-9A-F]{32}", g)
	}
}

func TestNewGUIDUniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		g := NewGUID()
		if seen[g] {
			t.Fatalf("NewGUID() produced duplicate: %s", g)
		}
		seen[g] = true
	}
}

func TestSeededGUIDDeterministic(t *testing.T) {
	fn1 := NewSeededGUIDFunc(42)
	fn2 := NewSeededGUIDFunc(42)

	for i := 0; i < 10; i++ {
		a, b := fn1(), fn2()
		if a != b {
			t.Fatalf("seeded GUID mismatch at call %d: %s vs %s", i, a, b)
		}
		if !guidPattern.MatchString(a) {
			t.Fatalf("seeded GUID %q doesn't match [0-9A-F]{32}", a)
		}
	}
}
