package app

import "context"

func ShutdownController(ctx context.Context, c Controller) error {
	if c == nil {
		return nil
	}
	return c.Shutdown(ctx)
}
