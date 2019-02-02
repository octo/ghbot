// Package changelog ensures that a pull request contians changelog information.
//
// Users may add `ChangeLog: [text]` to the pull request description. Text is a
// single change log entry along the lines of "Foo plugin: Implemented a
// thing.".
//
// For trivial changes, maintainers may set the "Unlisted Change" label which
// allows submission without change log information. If both are present, text
// and label, label wins out and now change log entry is being created.
package changelog

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/go-github/github"
	"github.com/octo/ghbot/client"
	"github.com/octo/ghbot/event"
	"google.golang.org/appengine/log"
)

const (
	checkName     = "ChangeLog"
	unlistedLabel = "Unlisted Change"
	detailsURL    = "https://github.com/collectd/collectd/blob/master/CONTRIBUTING.md#changelog"
)

var (
	logEntryRE = regexp.MustCompile(`(?m)^(?i:ChangeLog):\s*(\S.*)`)
)

func init() {
	event.PullRequestHandler("changelog", handler)
}

func formatEntry(pr *client.PR) (string, bool) {
	m := logEntryRE.FindStringSubmatch(pr.GetBody())
	if len(m) < 2 {
		return "", false
	}
	msg := m[1]

	user := pr.GetUser().GetName()
	if user == "" {
		user = "@" + pr.GetUser().GetLogin()
	}

	return fmt.Sprintf("%s Thanks to %s. %v", msg, user, pr), true
}

func handler(ctx context.Context, e *github.PullRequestEvent) error {
	triggerOn := map[string]bool{
		"edited":    true,
		"labeled":   true,
		"opened":    true,
		"unlabeled": true,
	}
	if !triggerOn[e.GetAction()] {
		return nil
	}

	c, err := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
	if err != nil {
		return err
	}

	pr := c.WrapPR(e.PullRequest)
	ref := pr.Head.GetSHA()
	log.Debugf(ctx, "checking if %v contains a changelog note", pr)

	// Only issues report the label :(
	i, err := pr.Issue(ctx)
	if err != nil {
		return err
	}

	if i.HasLabel(unlistedLabel) {
		return c.CreateStatus(ctx, checkName, client.StatusSuccess, "Pull request not included in ChangeLog", detailsURL, ref)
	}

	if entry, ok := formatEntry(pr); ok {
		msg := fmt.Sprintf("Preview: %q", entry)
		return c.CreateStatus(ctx, checkName, client.StatusSuccess, msg, detailsURL, ref)
	}

	return c.CreateStatus(ctx, checkName, client.StatusFailure, `Please add a "ChangeLog: …" line to your pull request description`, detailsURL, ref)
}
