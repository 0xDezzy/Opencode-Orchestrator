package issues

import "context"

type Tracker interface {
	FetchCandidateIssues(context.Context, FetchOptions) ([]Issue, error)
	FetchIssue(context.Context, string) (*Issue, error)
	Comment(context.Context, string, string) error
	Transition(context.Context, string, string) error
}
