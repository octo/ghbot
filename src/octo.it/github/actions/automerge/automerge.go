package automerge

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/github"
	"octo.it/github/client"
	"octo.it/github/event"
)

const automergeLabel = "Automerge"

func init() {
	event.PullRequestHandler(processPullRequestEvent)
	event.StatusHandler(processStatusEvent)
}

func processPullRequestEvent(ctx context.Context, event *github.PullRequestEvent) error {
	c := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
	return process(ctx, c.WrapPR(event.PullRequest))
}

func processStatusEvent(ctx context.Context, event *github.StatusEvent) error {
	if *event.State != "success" {
		return nil
	}

	c := client.New(ctx, client.DefaultOwner, client.DefaultRepo)

	pr, err := c.PullRequestBySHA(*event.SHA)
	if err != nil {
		log.Printf("PullRequestBySHA(%q) = %v", *event.SHA, err)
		return nil
	}

	return process(ctx, pr)
}

// process merges a pull request, if:
// * Is has not already been merged.
// * All three builders signal success.
// * There are no merge conflicts.
// * Is has the Automerge label.
func process(ctx context.Context, pr *client.PR) error {
	if pr.Merged == nil || *pr.Merged || pr.State == nil || *pr.State != "open" {
		return nil
	}

	status, err := pr.CombinedStatus()
	if err != nil {
		return err
	}

	// TODO(octo): find a better way to configure expected status entries.
	if len(status.Statuses) < 3 {
		return nil
	}

	if status.State == nil {
		log.Printf("PR %v has no status yes", pr)
		return nil
	} else if *status.State == "pending" {
		log.Printf(`PR %v has state "pending"`, pr)
		return nil
	}

	if *status.State != "success" {
		return nil
	}

	ok, err := pr.Mergeable()
	if err != nil {
		return err
	}
	if !ok {
		log.Printf("PR %v: checks succeeded but cannot merge", pr)
		return nil
	}

	issue, err := pr.Issue()
	if err != nil {
		return err
	}

	if !issue.HasLabel(automergeLabel) {
		return nil
	}

	title := fmt.Sprintf("Auto-Merge pull request %v from %s/%s", pr, *pr.Head.User.Login, *pr.Head.Ref)
	msg := fmt.Sprintf("Automatically merged due to %q label", automergeLabel)
	return pr.Merge(title, msg)
}
