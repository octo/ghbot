package automerge

import (
	"context"
	"fmt"

	"github.com/google/go-github/github"
	"github.com/octo/ghbot/client"
	"github.com/octo/ghbot/event"
	"google.golang.org/appengine/log"
)

const automergeLabel = "Automerge"

// requiredChecks is a list of all status "contexts" that must signal success
// before a PR can automatically be merged.
var requiredChecks = []string{
	"pull-requests-github_trigger-aggregation",
}

func init() {
	event.PullRequestHandler(processPullRequestEvent)
	event.StatusHandler(processStatusEvent)
}

func processPullRequestEvent(ctx context.Context, event *github.PullRequestEvent) error {
	c := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
	return process(ctx, c.WrapPR(event.PullRequest))
}

func processStatusEvent(ctx context.Context, event *github.StatusEvent) error {
	if *event.State != "success" {
		return nil
	}

	c := client.New(ctx, client.DefaultOwner, client.DefaultRepo)

	pr, err := c.PullRequestBySHA(ctx, *event.SHA)
	if err != nil {
		log.Debugf(ctx, "automerge: PullRequestBySHA(%q) = %v", *event.SHA, err)
		return nil
	}

	return process(ctx, pr)
}

// process merges a pull request, if:
// * Is has not already been merged.
// * All required checks have succeeded.
// * The overall state is "success".
// * There are no merge conflicts.
// * Is has the Automerge label.
func process(ctx context.Context, pr *client.PR) error {
	log.Debugf(ctx, "checking if %v can be automerged", pr)

	if pr.GetMerged() || pr.GetState() != "open" {
		log.Debugf(ctx, "automerge: no, not open")
		return nil
	}

	status, err := pr.CombinedStatus(ctx)
	if err != nil {
		return err
	}

	success := map[string]bool{}
	for _, s := range status.Statuses {
		success[s.GetContext()] = (s.GetState() == "success")
	}

	for _, req := range requiredChecks {
		if !success[req] {
			log.Debugf(ctx, "automerge: no, check %q missing or not successful", req)
			return nil
		}
	}

	if s := status.GetState(); s != "success" {
		log.Debugf(ctx, "automerge: no, overall status is %q", s)
		return nil
	}

	ok, err := pr.Mergeable(ctx)
	if err != nil {
		return err
	}
	if !ok {
		log.Debugf(ctx, "automerge: no, has merge conflicts")
		return nil
	}

	issue, err := pr.Issue(ctx)
	if err != nil {
		return err
	}

	if !issue.HasLabel(automergeLabel) {
		log.Debugf(ctx, "automerge: no, does not have the %q label", automergeLabel)
		return nil
	}

	log.Infof(ctx, "merging %v", pr)
	title := fmt.Sprintf("Auto-Merge pull request %v from %s/%s", pr, pr.Head.User.GetLogin(), pr.Head.GetRef())
	msg := fmt.Sprintf("Automatically merged due to %q label", automergeLabel)
	return pr.Merge(ctx, title, msg)
}
