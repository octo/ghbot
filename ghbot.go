package ghbot

import (
	"context"
	"fmt"
	"net/http"

	"cloud.google.com/go/trace"
	"github.com/google/go-github/github"
	"github.com/octo/ghbot/config"
	"github.com/octo/ghbot/event"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"

	_ "github.com/octo/ghbot/actions/automerge"
	_ "github.com/octo/ghbot/actions/format"
	_ "github.com/octo/ghbot/actions/milestone"
	_ "github.com/octo/ghbot/actions/newplugin"
)

func init() {
	http.HandleFunc("/", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	creds, err := google.FindDefaultCredentials(ctx, trace.ScopeTraceAppend)
	if err != nil {
		log.Criticalf(ctx, "google.FindDefaultCredentials(): %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	traceClient, err := trace.NewClient(ctx, creds.ProjectID,
		option.WithTokenSource(creds.TokenSource))
	if err != nil {
		log.Criticalf(ctx, "trace.NewClient(): %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	span := traceClient.SpanFromRequest(r)
	defer span.Finish()

	ctx = trace.NewContext(ctx, span)

	if r.Method != "POST" {
		http.Redirect(w, r, "https://github.com/collectd/collectd/", http.StatusFound)
		return
	}

	if err := contextHandler(ctx, w, r); err != nil {
		log.Errorf(ctx, "contextHandler: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func contextHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	secretKey, err := config.SecretKey(ctx)
	if err != nil {
		log.Errorf(ctx, "SecretKey: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	payload, err := github.ValidatePayload(r, secretKey)
	if err != nil {
		log.Errorf(ctx, "ValidatePayload: %v", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return nil
	}

	whType := github.WebHookType(r)
	if whType == "ping" {
		return processPing(ctx, w)
	}

	e, err := github.ParseWebHook(whType, payload)
	if err != nil {
		log.Errorf(ctx, "ParseWebHook: %v", err)
		httpStatusUnprocessableEntity := 422
		http.Error(w, err.Error(), httpStatusUnprocessableEntity)
		return nil
	}

	if err := event.Handle(ctx, e); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func processPing(ctx context.Context, w http.ResponseWriter) error {
	fmt.Fprintln(w, "pong")
	return nil
}
