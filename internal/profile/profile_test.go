package profile

import "testing"

func TestResolveBuiltins(t *testing.T) {
	tests := []string{"social-story", "podcast-clean", "archive-lossless"}
	for _, name := range tests {
		p, err := Resolve(name)
		if err != nil {
			t.Fatalf("Resolve(%s) failed: %v", name, err)
		}
		if p.Name == "" {
			t.Fatalf("Resolve(%s) returned empty profile name", name)
		}
	}
}

func TestResolveInvalid(t *testing.T) {
	_, err := Resolve("unknown-profile")
	if err == nil {
		t.Fatalf("expected error for unknown profile")
	}
}

func TestNamesCount(t *testing.T) {
	names := Names()
	if len(names) < 3 {
		t.Fatalf("expected at least 3 built-in profiles")
	}
}
