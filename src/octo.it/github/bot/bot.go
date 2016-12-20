package bot

import (
	"fmt"
	"net/http"

	"github.com/google/go-github/github"
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"octo.it/github/event"

	_ "octo.it/github/actions/automerge"
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
		log.Errorf(ctx, "contextHandler: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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

	e, err := github.ParseWebHook(whType, payload)
	if err != nil {
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
	log.Infof(ctx, "received ping")
	fmt.Fprintln(w, "pong")
	return nil
}
