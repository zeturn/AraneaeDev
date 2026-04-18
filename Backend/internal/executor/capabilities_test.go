package executor

import "testing"

func TestCollectRuntimeCapabilitiesContainsCoreKeys(t *testing.T) {
	caps := collectRuntimeCapabilities()
	if len(caps) == 0 {
		t.Fatal("expected at least one capability")
	}

	required := map[string]bool{
		"python": false,
		"node":   false,
		"go":     false,
		"java":   false,
	}
	for _, cap := range caps {
		if _, ok := required[cap.Key]; ok {
			required[cap.Key] = true
		}
	}
	for key, seen := range required {
		if !seen {
			t.Fatalf("missing capability key %q", key)
		}
	}
}

func TestFirstNonEmptyLine(t *testing.T) {
	if got := firstNonEmptyLine("\n  \npython 3.12.1\n"); got != "python 3.12.1" {
		t.Fatalf("unexpected first line: %q", got)
	}
}
