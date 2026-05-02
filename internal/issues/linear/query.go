package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"issue-orchestrator/internal/issues"
)

const candidateIssuesQuery = `query CandidateIssues($first: Int!, $after: String) {
  issues(first: $first, after: $after) {
    nodes {
      id
      identifier
      title
      description
      priority
      createdAt
      updatedAt
      number
      url
      branchName
      state { name }
      team { key }
      assignee { name displayName email }
      project { id name slugId }
      labels { nodes { name } }
    }
    pageInfo { hasNextPage endCursor }
  }
}`

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type graphQLResponse struct {
	Data   candidateIssuesData `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type candidateIssuesData struct {
	Issues candidateIssueConnection `json:"issues"`
}

type candidateIssueConnection struct {
	Nodes    []candidateIssue `json:"nodes"`
	PageInfo struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
}

type candidateIssue struct {
	ID          string      `json:"id"`
	Identifier  string      `json:"identifier"`
	Title       string      `json:"title"`
	Description *string     `json:"description"`
	Priority    float64     `json:"priority"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	Number      float64     `json:"number"`
	URL         string      `json:"url"`
	BranchName  string      `json:"branchName"`
	State       namedRef    `json:"state"`
	Team        teamRef     `json:"team"`
	Assignee    *userRef    `json:"assignee"`
	Project     *projectRef `json:"project"`
	Labels      struct {
		Nodes []namedRef `json:"nodes"`
	} `json:"labels"`
}

type namedRef struct {
	Name string `json:"name"`
}
type teamRef struct {
	Key string `json:"key"`
}
type userRef struct{ Name, DisplayName, Email string }
type projectRef struct{ ID, Name, SlugID string }

func (c *Client) fetchIssuePage(ctx context.Context, first int64, after *string) (*graphQLResponse, error) {
	vars := map[string]any{"first": first, "after": after}
	body, err := json.Marshal(graphQLRequest{Query: candidateIssuesQuery, Variables: vars})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, linearGraphQLEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "issue-orchestrator")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("linear candidate query failed: %s", resp.Status)
	}
	var out graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("linear candidate query failed: %s", out.Errors[0].Message)
	}
	return &out, nil
}

func (i candidateIssue) toIssue() issues.Issue {
	desc := ""
	if i.Description != nil {
		desc = *i.Description
	}
	assignee := ""
	if i.Assignee != nil {
		assignee = firstNonEmpty(i.Assignee.DisplayName, i.Assignee.Name, i.Assignee.Email)
	}
	project := ""
	if i.Project != nil {
		project = firstNonEmpty(i.Project.SlugID, i.Project.Name, i.Project.ID)
	}
	labels := make([]string, 0, len(i.Labels.Nodes))
	for _, label := range i.Labels.Nodes {
		if label.Name != "" {
			labels = append(labels, label.Name)
		}
	}
	return issues.Issue{ID: i.ID, Identifier: i.Identifier, Title: i.Title, Description: desc, URL: i.URL, State: i.State.Name, TeamKey: i.Team.Key, ProjectName: project, Priority: fmt.Sprint(i.Priority), Assignee: assignee, BranchName: i.BranchName, Labels: labels, CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt, Raw: i}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
