package ghbot

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/log"

	"github.com/google/go-github/github"
)

func processStatusEvent(ctx context.Context, event *github.StatusEvent) error {
	log.Infof(ctx, "event = %#v", event)
	return nil
}
