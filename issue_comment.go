package ghbot

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/log"

	"github.com/google/go-github/github"
)

func processIssueCommentEvent(ctx context.Context, event *github.IssueCommentEvent) error {
	log.Infof(ctx, "@%s %s a comment on #%d", *event.Issue.User.Login, *event.Action, *event.Issue.Number)
	return nil
}
