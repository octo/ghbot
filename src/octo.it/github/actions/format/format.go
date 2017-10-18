package format

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/google/go-github/github"
	"octo.it/github/client"
	"octo.it/github/event"
)

const (
	checkName   = "clang-format"
	clangFormat = "/usr/bin/clang-format"
)

func init() {
	event.PullRequestHandler(processPullRequestEvent)
}

func hasAnySuffix(s string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}

	return false
}

func processPullRequestEvent(ctx context.Context, e *github.PullRequestEvent) error {
	if a := e.GetAction(); a != "opened" && a != "synchronize" {
		return nil
	}

	c := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
	pr := c.WrapPR(e.PullRequest)

	ref := pr.Head.GetSHA()
	if err := c.CreateStatus(ctx, checkName, client.StatusPending, "Checking formatting ...", ref); err != nil {
		return err
	}

	handleError := func(err error) error {
		msg := fmt.Sprintf("clang-format failed: %v", err)
		c.CreateStatus(ctx, checkName, client.StatusError, msg, ref)
		return err
	}

	files, err := pr.Files(ctx)
	if err != nil {
		return err
	}

	var total int
	var needFormatting []string
	for _, f := range files {
		if !hasAnySuffix(f.Filename, []string{".c", ".h", ".proto"}) {
			continue
		}

		content, err := pr.Blob(ctx, f.SHA)
		if err != nil {
			return handleError(err)
		}

		ok, err := checkFormat(ctx, content)
		if err != nil {
			return handleError(err)
		}

		if !ok {
			needFormatting = append(needFormatting, f.Filename)
		}
		total++
	}

	if total == 0 {
		return c.CreateStatus(ctx, checkName, client.StatusSuccess, "No matching files", ref)
	}

	if len(needFormatting) != 0 {
		sort.Strings(needFormatting)
		msg := "Please fix formatting: clang-format -style=file -i " + strings.Join(needFormatting, " ")
		return c.CreateStatus(ctx, checkName, client.StatusFailure, msg, ref)
	}

	// all files are well formatted
	msg := "File is correctly formatted"
	if total != 1 {
		msg = fmt.Sprintf("%d files are correctly formatted", total)
	}
	return c.CreateStatus(ctx, checkName, client.StatusSuccess, msg, ref)
}

func checkFormat(ctx context.Context, in string) (bool, error) {
	cmd := exec.CommandContext(ctx, clangFormat, "-style=LLVM")
	cmd.Stdin = strings.NewReader(in)

	out := &bytes.Buffer{}
	cmd.Stdout = out

	errbuf := &bytes.Buffer{}
	cmd.Stderr = errbuf

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("clang-format: %v\nSTDERR: %s", err, errbuf)
	}

	return in == out.String(), nil
}
