package format

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/google/go-github/github"
	"github.com/octo/ghbot/client"
	"github.com/octo/ghbot/event"
	"go.opencensus.io/plugin/ochttp"
)

const (
	checkName  = "clang-format"
	formatURL  = "https://format.collectd.org/"
	detailsURL = ""
)

var (
	suffixes = []string{
		".c",
		".cc",
		".h",
		".java",
		".proto",
	}
)

func init() {
	event.PullRequestHandler("format", processPullRequestEvent)
}

func hasAnySuffix(s string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}

	return false
}

type checkFileStatus struct {
	ok  bool
	err error

	client.PRFile
}

func processPullRequestEvent(ctx context.Context, e *github.PullRequestEvent) error {
	if a := e.GetAction(); a != "opened" && a != "synchronize" {
		return nil
	}

	c, err := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
	if err != nil {
		return err
	}

	pr := c.WrapPR(e.PullRequest)
	ref := pr.Head.GetSHA()

	files, err := pr.Files(ctx)
	if err != nil {
		return err
	}

	stage := c.NewStage(e.PullRequest)
	ch := make(chan checkFileStatus)
	wg := &sync.WaitGroup{}

	var total int
	for _, f := range files {
		if !hasAnySuffix(f.Filename, suffixes) {
			continue
		}

		total++
		wg.Add(1)

		// Pass f as argument so it is being copied, i.e. f inside the
		// closure is a different variable than the loop variable,
		// which will be changed soon, causing a race condition.
		go func(f client.PRFile) {
			ok, err := checkFile(ctx, pr, f, stage)
			ch <- checkFileStatus{
				ok:     ok,
				err:    err,
				PRFile: f,
			}
			wg.Done()
		}(f)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	if total == 0 {
		return c.CreateStatus(ctx, checkName, client.StatusSuccess, "PR contains no affected files", detailsURL, ref)
	}

	if err := c.CreateStatus(ctx, checkName, client.StatusPending, "Checking formatting ...", detailsURL, ref); err != nil {
		return err
	}

	var needFormatting []string

	err = nil
	for s := range ch {
		if s.err != nil {
			err = s.err
			continue
		}

		if !s.ok {
			needFormatting = append(needFormatting, s.Filename)
		}
	}

	if err != nil {
		c.CreateStatus(ctx, checkName, client.StatusError, err.Error(), detailsURL, ref)
		return err
	}

	if len(needFormatting) == 0 {
		msg := "File is correctly formatted"
		if total != 1 {
			msg = fmt.Sprintf("%d files are correctly formatted", total)
		}
		return c.CreateStatus(ctx, checkName, client.StatusSuccess, msg, detailsURL, ref)
	}

	sort.Strings(needFormatting)
	msg := "Please run: contrib/format.sh " + strings.Join(needFormatting, " ")

	if err := c.CreateStatus(ctx, checkName, client.StatusFailure, msg, detailsURL, ref); err != nil {
		return err
	}

	// TODO(octo): this does not work, because the Github API doesn't allow
	// us to create blobs or trees in the reposiroty of the PR author (it returns a 404).
	// Either figure out how to do this, create the changes in our own repo
	// and create a PR (ideal, but how to pull in all changes?) or give up
	// on the idea.
	/*
		if pr.GetMaintainerCanModify() {
			if err := stage.Commit(ctx, msg); err != nil {
				return err
			}
		}
	*/

	return nil
}

func checkFile(ctx context.Context, pr *client.PR, f client.PRFile, stage *client.Stage) (bool, error) {
	got, err := pr.Blob(ctx, f.SHA)
	if err != nil {
		return false, err
	}

	want, err := format(ctx, got)
	if err != nil {
		return false, err
	}

	if got != want {
		stage.Add(f.Filename, want)
		return false, nil
	}

	return true, nil
}

func format(ctx context.Context, in string) (string, error) {
	// matches clang-format-gae's timeout.
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client := &http.Client{
		Transport: &ochttp.Transport{
			Propagation: &propagation.HTTPFormat{},
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, formatURL, strings.NewReader(in))
	if err != nil {
		return "", fmt.Errorf("NewRequestWithContext(): %v", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %s", res.Status)
	}

	out, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(out), nil
}
