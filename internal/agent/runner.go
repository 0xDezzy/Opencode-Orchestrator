package agent

import "context"

type Runner interface {
	RunIssue(context.Context, RunIssueRequest) (*RunIssueResult, error)
}
