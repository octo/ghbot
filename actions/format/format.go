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

	"github.com/google/go-github/github"
	"github.com/octo/ghbot/client"
	"github.com/octo/ghbot/event"
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

type checkFileStatus struct {
	ok  bool
	err error

	client.PRFile
}

func processPullRequestEvent(ctx context.Context, e *github.PullRequestEvent) error {
	if a := e.GetAction(); a != "opened" && a != "synchronize" {
		return nil
	}

	c := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
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
		if !hasAnySuffix(f.Filename, []string{".c", ".h", ".proto"}) {
			continue
		}

		total++
		wg.Add(1)
		go func() {
			ok, err := checkFile(ctx, pr, f, stage)
			ch <- checkFileStatus{
				ok:     ok,
				err:    err,
				PRFile: f,
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	if total == 0 {
		return nil
	}

	if err := c.CreateStatus(ctx, checkName, client.StatusPending, "Checking formatting ...", ref); err != nil {
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
		c.CreateStatus(ctx, checkName, client.StatusError, err.Error(), ref)
		return err
	}

	if len(needFormatting) == 0 {
		msg := "File is correctly formatted"
		if total != 1 {
			msg = fmt.Sprintf("%d files are correctly formatted", total)
		}
		return c.CreateStatus(ctx, checkName, client.StatusSuccess, msg, ref)
	}

	msg := "clang-format -style=file -i " + strings.Join(needFormatting, " ")

	sort.Strings(needFormatting)
	if err := c.CreateStatus(ctx, checkName, client.StatusFailure, "Please fix formatting: "+msg, ref); err != nil {
		return err
	}

	if pr.GetMaintainerCanModify() {
		if err := stage.Commit(ctx, msg); err != nil {
			return err
		}
	}

	return nil
}

func checkFile(ctx context.Context, pr *client.PR, f client.PRFile, stage *client.Stage) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

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
	req, err := http.NewRequest(http.MethodPost, "https://clang-format.appspot.com/", strings.NewReader(in))
	if err != nil {
		return "", err
	}
	req = req.WithContext(ctx)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	out, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(out), nil
}