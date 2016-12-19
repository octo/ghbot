package ghbot

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/log"

	"github.com/google/go-github/github"
)

func processPullRequestEvent(ctx context.Context, event *github.PullRequestEvent) error {
	log.Infof(ctx, "@%s %s pull request #%d", *event.Sender.Login, *event.Action, *event.Number)
	return nil
}
