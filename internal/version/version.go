package version

import (
	"os/exec"
	"strings"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

var describeVersion = gitDescribeVersion

func String() string {
	version := Version
	if version == "dev" {
		if described, err := describeVersion(); err == nil && described != "" {
			version = described
		}
	}

	return "orchestrator " + version + " (commit " + Commit + ", built " + Date + ")"
}

func gitDescribeVersion() (string, error) {
	out, err := exec.Command("git", "describe", "--tags", "--dirty", "--always").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
