package linear

import (
	"context"
	"fmt"
	"net/http"

	lin "github.com/chainguard-sandbox/go-linear/v2/pkg/linear"

	"issue-orchestrator/internal/common/config"
	"issue-orchestrator/internal/issues"
)

const linearGraphQLEndpoint = "https://api.linear.app/graphql"

type Client struct {
	c      *lin.Client
	apiKey string
	http   *http.Client
}

func New(cfg config.LinearConfig) (*Client, error) {
	c, err := lin.NewClient(cfg.APIKey)
	if err != nil {
		return nil, err
	}
	return &Client{c: c, apiKey: cfg.APIKey, http: http.DefaultClient}, nil
}
func (c *Client) FetchCandidateIssues(ctx context.Context, o issues.FetchOptions) ([]issues.Issue, error) {
	first := int64(o.Limit)
	if first <= 0 {
		first = 50
	}
	out := []issues.Issue{}
	pageSize := int64(50)
	if first < pageSize {
		pageSize = first
	}
	var after *string
	for scanned := int64(0); scanned < 500 && int64(len(out)) < first; {
		res, err := c.fetchIssuePage(ctx, pageSize, after)
		if err != nil {
			return nil, err
		}
		for _, raw := range res.Data.Issues.Nodes {
			scanned++
			iss := raw.toIssue()
			if eligible(iss, o) {
				out = append(out, iss)
				if int64(len(out)) >= first {
					break
				}
			}
		}
		if !res.Data.Issues.PageInfo.HasNextPage || res.Data.Issues.PageInfo.EndCursor == "" {
			break
		}
		after = &res.Data.Issues.PageInfo.EndCursor
	}
	return out, nil
}
func (c *Client) FetchIssue(ctx context.Context, id string) (*issues.Issue, error) {
	raw, err := c.c.Issue(ctx, id)
	if err == nil {
		iss := mapAnyIssue(raw)
		return &iss, nil
	}
	results, serr := c.c.SearchIssues(ctx, id, nil, nil, nil, nil)
	if serr != nil {
		return nil, fmt.Errorf("fetch issue %s: %w", id, err)
	}
	for _, raw := range reflectNodes(results) {
		iss := mapAnyIssue(raw)
		if iss.Identifier == id || iss.ID == id {
			return &iss, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", issues.ErrIssueNotFound, id)
}
func (c *Client) Comment(ctx context.Context, issueID, body string) error {
	if err := validateMutationTarget(issueID); err != nil {
		return err
	}
	in := lin.CommentCreateInput{IssueID: &issueID, Body: &body}
	_, err := c.c.CommentCreate(ctx, in)
	return err
}
func (c *Client) Transition(ctx context.Context, issueID, stateName string) error {
	if err := validateMutationTarget(issueID); err != nil {
		return err
	}
	if err := validateTransitionState(stateName); err != nil {
		return err
	}
	states, err := c.c.WorkflowStates(ctx, nil, nil)
	if err != nil {
		return err
	}
	for _, raw := range reflectNodes(states) {
		st := mapState(raw)
		if st.name == stateName {
			in := lin.IssueUpdateInput{StateID: &st.id}
			_, err := c.c.IssueUpdate(ctx, issueID, in)
			return err
		}
	}
	return fmt.Errorf("linear state %q not found", stateName)
}
