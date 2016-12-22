package automerge

import (
	"context"
	"log"

	"github.com/google/go-github/github"
	"octo.it/github/client"
	"octo.it/github/event"
)

func init() {
	event.PullRequestHandler(processPullRequestEvent)
	event.StatusHandler(processStatusEvent)
}

func processPullRequestEvent(ctx context.Context, event *github.PullRequestEvent) error {
	pr := event.PullRequest

	if pr.State == nil || *pr.State != "open" {
		return nil
	}

	return automerge(ctx, pr)
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

	return automerge(ctx, pr)
}

func automerge(ctx context.Context, pr *github.PullRequest) error {
	log.Printf("automerge(%v)", pr)
	return nil
}
