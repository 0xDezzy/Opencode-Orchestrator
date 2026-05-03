package version

import (
	"errors"
	"testing"
)

func TestStringUsesInjectedVersion(t *testing.T) {
	withVersionState(t, "v1.2.3", "abc123", "2026-05-02T00:00:00Z", func() (string, error) {
		t.Fatal("describeVersion should not be called for injected versions")
		return "", nil
	})

	got := String()
	want := "orchestrator v1.2.3 (commit abc123, built 2026-05-02T00:00:00Z)"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestStringUsesGitDescribeForDevVersion(t *testing.T) {
	withVersionState(t, "dev", "unknown", "unknown", func() (string, error) {
		return "v1.2.3-4-gabc123-dirty", nil
	})

	got := String()
	want := "orchestrator v1.2.3-4-gabc123-dirty (commit unknown, built unknown)"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestStringKeepsDevWhenGitDescribeFails(t *testing.T) {
	withVersionState(t, "dev", "unknown", "unknown", func() (string, error) {
		return "", errors.New("git unavailable")
	})

	got := String()
	want := "orchestrator dev (commit unknown, built unknown)"
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func withVersionState(t *testing.T, version, commit, date string, describe func() (string, error)) {
	t.Helper()

	originalVersion := Version
	originalCommit := Commit
	originalDate := Date
	originalDescribeVersion := describeVersion

	Version = version
	Commit = commit
	Date = date
	describeVersion = describe

	t.Cleanup(func() {
		Version = originalVersion
		Commit = originalCommit
		Date = originalDate
		describeVersion = originalDescribeVersion
	})
}
