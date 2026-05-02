package git

import "testing"

func TestSanitizeAndContainment(t *testing.T) {
	if got := SanitizePart("../Bad Title!"); got == "" || got == ".." {
		t.Fatalf("unsafe slug %q", got)
	}
	if !Inside("/tmp/root", "/tmp/root/a/b") {
		t.Fatal("expected child inside root")
	}
	if Inside("/tmp/root", "/tmp/root2/a") {
		t.Fatal("sibling must not be inside root")
	}
}
