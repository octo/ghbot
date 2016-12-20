package automerge

import (
	"github.com/google/go-github/github"
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
	"octo.it/github/event"
)

func init() {
	event.StatusHandler(processStatusEvent)
}

func processStatusEvent(ctx context.Context, event *github.StatusEvent) error {
	log.Infof(ctx, "event = %#v", event)
	return nil
}
