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
	"google.golang.org/appengine/log"
)

func init() {
	event.PullRequestHandler("milestone", handler)
}

func handler(ctx context.Context, e *github.PullRequestEvent) error {
	if a := e.GetAction(); a != "opened" && a != "edited" {
		return nil
	}

	c := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
	pr := c.WrapPR(e.PullRequest)

	log.Debugf(ctx, "checking if a milestone should be set for %v", pr)

	ref := pr.PullRequest.Base.GetRef()

	// This is likely a PR for the master branch.
	if !strings.HasPrefix(ref, "collectd-") {
		log.Debugf(ctx, "milestone: no, not a feature branch: %q", ref)
		return nil
	}
	v := strings.TrimPrefix(ref, "collectd-")

	// Only issues report the milestone :(
	i, err := pr.Issue(ctx)
	if err != nil {
		return err
	}

	// A milestone has already been set.
	if i.Issue.Milestone != nil {
		log.Debugf(ctx, "milestone: no, already set to %q", i.Issue.Milestone.GetTitle())
		return nil
	}

	milestones, err := c.Milestones(ctx)
	if err != nil {
		return err
	}

	for title, id := range milestones {
		if title == v {
			return i.Milestone(ctx, id)
		}
	}

	log.Debugf(ctx, "milestone: no, milestone %q not found", v)
	return nil
}
