package automerge

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/go-github/github"
	"github.com/octo/ghbot/client"
	"github.com/octo/ghbot/event"
	"go.uber.org/multierr"
	"google.golang.org/appengine/log"
)

const (
	newLabel         = "New plugin"
	defaultMilestone = "Features"
)

var requiredFiles = []string{
	"README",
	"src/collectd.conf.pod",
	"src/collectd.conf.in",
}

func init() {
	event.PullRequestHandler("newplugin", processPullRequestEvent)
}

func processPullRequestEvent(ctx context.Context, event *github.PullRequestEvent) error {
	c, err := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
	if err != nil {
		return err
	}

	return process(ctx, c, c.WrapPR(event.PullRequest))
}

func process(ctx context.Context, c *client.Client, pr *client.PR) error {
	if pr.GetMerged() || pr.GetState() != "open" {
		return nil
	}

	issue, err := pr.Issue(ctx)
	if err != nil {
		return err
	}

	if !issue.HasLabel(newLabel) {
		return nil
	}

	wg := sync.WaitGroup{}
	ch := make(chan error)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := setMilestone(ctx, c, issue); err != nil {
			ch <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := checkFiles(ctx, c, pr); err != nil {
			ch <- err
		}
	}()

	go func() {
		wg.Wait()
		close(ch)
	}()

	var errs error
	for e := range ch {
		errs = multierr.Append(errs, e)
	}

	return errs
}

func setMilestone(ctx context.Context, c *client.Client, issue *client.Issue) error {
	if issue.GetMilestone() != nil {
		return nil
	}

	milestones, err := c.Milestones(ctx)
	if err != nil {
		return fmt.Errorf("newplugin: %v", err)
	}

	id, ok := milestones[defaultMilestone]
	if !ok {
		log.Warningf(ctx, "unable to determine ID of milestone %q", defaultMilestone)
		return nil
	}

	return issue.Milestone(ctx, id)
}

func checkFiles(ctx context.Context, c *client.Client, pr *client.PR) error {
	files, err := pr.Files(ctx)
	if err != nil {
		return fmt.Errorf("newplugin: %v", err)
	}

	got := map[string]struct{}{}
	for _, f := range files {
		got[f.Filename] = struct{}{}
	}

	var want []string
	for _, f := range requiredFiles {
		if _, ok := got[f]; ok {
			continue
		}

		want = append(want, f)
	}

	name := "new-plugin-docs"
	ref := pr.GetHead().GetSHA()
	status := client.StatusSuccess
	msg := "All required files touched"
	if len(want) != 0 {
		status = client.StatusFailure
		msg = "Document new plugin in: " + strings.Join(want, ", ")
	}

	return c.CreateStatus(ctx, name, status, msg, ref)
}
