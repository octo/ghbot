// Package changelog ensures that a pull request contians changelog information.
//
// Users may add `ChangeLog: [text]` to the pull request description. Text is a
// single change log entry along the lines of "Foo plugin: Implemented a
// thing.".
//
// For trivial changes, maintainers may set the "Unlisted Change" label which
// allows submission without change log information. If both are present, text
// and label, label wins out and now change log entry is being created.
package labels

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"bitbucket.org/creachadair/stringset"
	"github.com/google/go-github/github"
	"github.com/octo/ghbot/client"
	"github.com/octo/ghbot/event"
)

const (
	checkName  = "Labels"
	detailsURL = "https://github.com/collectd/collectd/blob/master/docs/CONTRIBUTING.md#changelog"
)

const (
	labelFeature     = "Feature"
	labelFix         = "Fix"
	labelMaintenance = "Maintenance"
)

var requiredLabels = stringset.New(labelFix, labelFeature, labelMaintenance)

func init() {
	event.PullRequestHandler("labels", handler)
}

func handler(ctx context.Context, e *github.PullRequestEvent) error {
	triggerOn := map[string]bool{
		"edited":      true,
		"labeled":     true,
		"opened":      true,
		"synchronize": true,
		"unlabeled":   true,
	}
	if !triggerOn[e.GetAction()] {
		return nil
	}

	var gotLabels stringset.Set
	for _, label := range e.GetPullRequest().Labels {
		gotLabels.Add(label.GetName())
	}

	c, err := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
	if err != nil {
		return err
	}

	pr := c.WrapPR(e.GetPullRequest())
	ref := pr.Head.GetSHA()

	relevantLabels := gotLabels.Intersect(requiredLabels)
	if relevantLabels.Len() == 1 {
		return c.CreateStatus(ctx, checkName, client.StatusSuccess,
			fmt.Sprintf("The PR is marked as %q", relevantLabels.Unordered()[0]),
			detailsURL, ref)
		return nil
	}

	if relevantLabels.Len() > 1 {
		return c.CreateStatus(ctx, checkName, client.StatusFailure,
			fmt.Sprintf("The labels %q are mutually exclusive. Pick one.", relevantLabels.Elements()),
			detailsURL, ref)
	}

	if label, ok := guessLabel(pr); ok {
		issue, err := pr.Issue(ctx)
		if err != nil {
			return err
		}

		if err := issue.AddLabel(ctx, label); err != nil {
			return err
		}

		return c.CreateStatus(ctx, checkName, client.StatusSuccess,
			fmt.Sprintf("Guessing this is a %q", label),
			detailsURL, ref)
	}

	return c.CreateStatus(ctx, checkName, client.StatusFailure,
		fmt.Sprintf("One of %q has to be set.", requiredLabels.Elements()),
		detailsURL, ref)
}

func guessLabel(pr *client.PR) (string, bool) {
	prefixToLabel := map[string]string{
		"feat":     labelFeature,
		"fix":      labelFix,
		"build":    labelMaintenance,
		"chore":    labelMaintenance,
		"ci":       labelMaintenance,
		"docs":     labelFix,
		"style":    labelMaintenance,
		"refactor": labelMaintenance,
		"perf":     labelFeature,
		"test":     labelMaintenance,
	}
	re := regexp.MustCompile(`^(feat|fix|build|chore|ci|docs|style|refactor|perf|test)\b`)

	title := strings.TrimPrefix(pr.GetTitle(), "[collectd 6] ")

	prefix := re.FindString(title)

	label, ok := prefixToLabel[prefix]
	return label, ok
}
