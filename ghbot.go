package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/trace"
	"github.com/google/go-github/github"
	"github.com/octo/ghbot/config"
	"github.com/octo/ghbot/event"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	_ "github.com/octo/ghbot/actions/automerge"
	_ "github.com/octo/ghbot/actions/changelog"
	_ "github.com/octo/ghbot/actions/format"
	_ "github.com/octo/ghbot/actions/milestone"
	_ "github.com/octo/ghbot/actions/newplugin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalln("http.ListenAndServe:", err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	creds, err := google.FindDefaultCredentials(ctx, trace.ScopeTraceAppend)
	if err != nil {
		log.Println("google.FindDefaultCredentials():", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	traceClient, err := trace.NewClient(ctx, creds.ProjectID,
		option.WithTokenSource(creds.TokenSource))
	if err != nil {
		log.Println("trace.NewClient():", err)
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
		log.Println("contextHandler:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func contextHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	secretKey, err := config.SecretKey(ctx)
	if err != nil {
		log.Println("SecretKey:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	payload, err := github.ValidatePayload(r, secretKey)
	if err != nil {
		log.Println("ValidatePayload:", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return nil
	}

	whType := github.WebHookType(r)
	if whType == "ping" {
		return processPing(ctx, w)
	}

	e, err := github.ParseWebHook(whType, payload)
	if err != nil {
		log.Println("ParseWebHook:", err)
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
