package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/google/go-github/github"
	"octo.it/github/event"

	_ "octo.it/github/actions/automerge"
	_ "octo.it/github/actions/format"
)

var secretKey = []byte("@SECRET@")

func main() {
	http.HandleFunc("/_ah/health", healthCheckHandler)
	http.HandleFunc("/", handler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != "POST" {
		http.Redirect(w, r, "https://github.com/collectd/collectd/", http.StatusFound)
		return
	}

	if err := contextHandler(ctx, w, r); err != nil {
		log.Printf("contextHandler: %v", err)
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
	log.Printf("received ping")
	fmt.Fprintln(w, "pong")
	return nil
}

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "ok")
}
