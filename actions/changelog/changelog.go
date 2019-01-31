// Package changelog ensures that a pull request contians changelog information.
//
// Users may add `ChangeLog=[text]` to the pull request description. Text is a
// single change log entry along the lines of "Foo plugin: Implemented a
// thing.".
//
// For trivial changes, maintainers may set the "Unlisted Change" label which
// allows submission without change log information. If both are present, text
// and label, label wins out and now change log entry is being created.
package changelog

import (
	"context"
	"regexp"

	"github.com/google/go-github/github"
	"github.com/octo/ghbot/client"
	"github.com/octo/ghbot/event"
	"google.golang.org/appengine/log"
)

const (
	checkName     = "ChangeLog"
	unlistedLabel = "Unlisted Change"
)

var (
	logEntryRE = regexp.MustCompile(`^(?i:ChangeLog)\s*=\s*(\S.*)`)
)

func init() {
	event.PullRequestHandler("changelog", handler)
}

func hasLogEntry(pr *client.PR) bool {
	return logEntryRE.MatchString(pr.GetBody())
}

func handler(ctx context.Context, e *github.PullRequestEvent) error {
	if a := e.GetAction(); a != "opened" && a != "edited" {
		return nil
	}

	c, err := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
	if err != nil {
		return err
	}

	pr := c.WrapPR(e.PullRequest)
	log.Debugf(ctx, "checking if %v contains a changelog note", pr)

	// Only issues report the label :(
	i, err := pr.Issue(ctx)
	if err != nil {
		return err
	}

	ref := pr.Head.GetSHA()

	if i.HasLabel(unlistedLabel) {
		return c.CreateStatus(ctx, checkName, client.StatusSuccess, "Pull request not included in ChangeLog", ref)
	}
	if hasLogEntry(pr) {
		// TODO(octo): Maybe echo the parsed information back to the user?
		return c.CreateStatus(ctx, checkName, client.StatusSuccess, "ChangeLog information found", ref)
	}

	return c.CreateStatus(ctx, checkName, client.StatusFailure, `Please add a "ChangeLog=..." line to your pull request description`, ref)
}
