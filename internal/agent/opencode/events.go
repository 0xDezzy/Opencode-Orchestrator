package opencode

import (
	"context"
	"fmt"

	acp "github.com/coder/acp-go-sdk"
)

type client struct {
	acp.Client
	events func(string, string)
}

func (c client) SessionUpdate(ctx context.Context, p acp.SessionNotification) error {
	if c.events != nil {
		c.events("session_update", fmt.Sprintf("%T", p.Update))
	}
	return nil
}
