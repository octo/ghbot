package ghbot

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/log"

	"github.com/google/go-github/github"
)

func processIssuesEvent(ctx context.Context, event *github.IssuesEvent) error {
	log.Infof(ctx, "@%s %s issue #%d", *event.Sender.Login, *event.Action, *event.Issue.Number)
	return nil
}
