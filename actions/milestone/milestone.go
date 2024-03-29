// Package milestone sets the milestone of a PR based on the base branch.
//
// Branches are expected to be named "collectd-<major>.<minor>", milestones are
// expected to be titled "<major>.<minor>".
package milestone

import (
	"context"
	"strings"

	"github.com/google/go-github/github"
	"github.com/octo/ghbot/client"
	"github.com/octo/ghbot/event"
)

func init() {
	event.PullRequestHandler("milestone", handler)
}

func handler(ctx context.Context, e *github.PullRequestEvent) error {
	if a := e.GetAction(); a != "opened" && a != "edited" {
		return nil
	}

	c, err := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
	if err != nil {
		return err
	}

	pr := c.WrapPR(e.PullRequest)

	ref := pr.PullRequest.Base.GetRef()

	// This is likely a PR for the main branch.
	if !strings.HasPrefix(ref, "collectd-") {
		return nil
	}
	version := strings.TrimPrefix(ref, "collectd-")

	// Only issues report the milestone :(
	i, err := pr.Issue(ctx)
	if err != nil {
		return err
	}

	// A milestone has already been set.
	if i.Issue.Milestone != nil {
		return nil
	}

	milestones, err := c.Milestones(ctx)
	if err != nil {
		return err
	}

	if id, ok := milestones[version]; ok {
		return i.Milestone(ctx, id)
	}

	return nil
}
