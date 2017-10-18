package format

import (
	"bytes"
	"context"
	"log"
	"os/exec"

	"github.com/google/go-github/github"
	"octo.it/github/client"
	"octo.it/github/event"
)

const checkName = "clang-format"

func init() {
	event.PullRequestHandler(processPullRequestEvent)
}

func processPullRequestEvent(ctx context.Context, e *github.PullRequestEvent) error {
	if a := e.GetAction(); a != "opened" && a != "synchronize" {
		return nil
	}

	c := client.New(ctx, client.DefaultOwner, client.DefaultRepo)

	ref := e.PullRequest.Head.GetSHA()
	if err := c.CreateStatus(ctx, checkName, client.StatusPending, "Checking formatting ...", ref); err != nil {
		return err
	}

	// url := e.PullRequest.Repository.GitURL()
	owner := e.PullRequest.Head.Repo.Owner.GetLogin()
	repo := e.PullRequest.Head.Repo.GetName()
	branch := e.PullRequest.Head.GetRef()
	base := e.PullRequest.Base.GetRef()

	cmd := exec.CommandContext(ctx, "/opt/format-bot/bin/check_formatting.sh", owner, repo, branch, base)
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}

	err := cmd.Run()
	if err == nil {
		return c.CreateStatus(ctx, checkName, client.StatusSuccess, "All files formatted correctly", ref)
	}

	if _, ok := err.(*exec.ExitError); ok {
		log.Printf("check_formatting.sh failed: %v\n%q", err, cmd.Stdout.(*bytes.Buffer).String())
		return c.CreateStatus(ctx, checkName, client.StatusFailure, "Please run clang-format on all modified files", ref)
	}

	return c.CreateStatus(ctx, checkName, client.StatusError, err.Error(), ref)
}
