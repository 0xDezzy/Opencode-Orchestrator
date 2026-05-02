package version

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func String() string {
	return "orchestrator " + Version + " (commit " + Commit + ", built " + Date + ")"
}
