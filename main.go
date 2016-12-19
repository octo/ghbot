package ghbot

import (
	"fmt"
	"net/http"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"

	"github.com/google/go-github/github"
)

var secretKey = []byte("@SECRET@")

func init() {
	http.HandleFunc("/", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	if r.Method != "POST" {
		http.Redirect(w, r, "https://github.com/collectd/collectd/", http.StatusFound)
		return
	}

	if err := contextHandler(ctx, w, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Errorf(ctx, "contextHandler: %v", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func contextHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	payload, err := github.ValidatePayload(r, secretKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return nil
	}

	whType := github.WebHookType(r)
	if whType == "ping" {
		return processPing(ctx, w)
	}

	event, err := github.ParseWebHook(whType, payload)
	if err != nil {
		httpStatusUnprocessableEntity := 422
		http.Error(w, err.Error(), httpStatusUnprocessableEntity)
		return nil
	}
	switch event := event.(type) {
	case *github.IssueCommentEvent:
		return processIssueCommentEvent(ctx, event)
	case *github.IssuesEvent:
		return processIssuesEvent(ctx, event)
	case *github.PullRequestEvent:
		return processPullRequestEvent(ctx, event)
	default:
		log.Debugf(ctx, "unimplemented event type: %T", event)
	}

	return nil
}

func processPing(ctx context.Context, w http.ResponseWriter) error {
	log.Infof(ctx, "received ping")
	fmt.Fprintln(w, "pong")
	return nil
}
