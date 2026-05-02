package issues

import "time"

type Issue struct {
	ID, Identifier, Title, Description, URL, State, TeamKey, ProjectName, Priority, Assignee, BranchName string
	Labels                                                                                               []string
	Raw                                                                                                  any
	CreatedAt, UpdatedAt                                                                                 time.Time
}
type FetchOptions struct {
	TeamKey, ProjectName                       string
	ActiveStates, IncludeLabels, ExcludeLabels []string
	Limit                                      int
}
