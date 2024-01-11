package client

import (
	"context"
	"fmt"

	"github.com/google/go-github/github"
)

func (c *Client) CheckRuns(ctx context.Context, ref string) ([]*github.CheckRun, error) {
	opts := github.ListCheckRunsOptions{}
	resp, _, err := c.Client.Checks.ListCheckRunsForRef(ctx, c.owner, c.repo, ref, &opts)
	if err != nil {
		return nil, fmt.Errorf("ListCheckRunsForRef(%q, %q, %q): %w", c.owner, c.repo, ref, err)
	}

	return resp.CheckRuns, nil
}
