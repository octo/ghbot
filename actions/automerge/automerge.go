package automerge

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/github"
	"github.com/octo/gaelog"
	"github.com/octo/ghbot/client"
	"github.com/octo/ghbot/event"
)

const automergeLabel = "Automerge"

// requiredChecks is a list of all status "contexts" that must signal success
// before a PR can automatically be merged.
// TODO(octo): may be redundant with the "Require status checks to pass before merging" setting.
var requiredChecks = []string{
	"ChangeLog",
	"continuous-integration/travis-ci/pr",
}

func init() {
	event.PullRequestHandler("automerge", processPullRequestEvent)
	event.PullRequestReviewHandler("automerge", processReviewEvent)
	event.StatusHandler("automerge", processStatusEvent)
}

func processPullRequestEvent(ctx context.Context, event *github.PullRequestEvent) error {
	c, err := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
	if err != nil {
		return err
	}

	return process(ctx, c.WrapPR(event.PullRequest))
}

func processReviewEvent(ctx context.Context, e *github.PullRequestReviewEvent) error {
	c, err := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
	if err != nil {
		return err
	}

	return process(ctx, c.WrapPR(e.GetPullRequest()))
}

func processStatusEvent(ctx context.Context, event *github.StatusEvent) error {
	if event.GetState() != "success" {
		return nil
	}

	c, err := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
	if err != nil {
		return err
	}

	pr, err := c.PullRequestBySHA(ctx, event.GetSHA())
	if err == os.ErrNotExist {
		gaelog.Debugf(ctx, "automerge: no pull request found for %s", event.GetSHA())
		return nil
	} else if err != nil {
		return err
	}

	return process(ctx, pr)
}

// process merges a pull request, if:
// * It it still open and has not already been merged.
// * Is has the Automerge label.
// * There are no outstanding reviews (CHANGES_REQUESTED or PENDING).
// * The overall state is "success".
// * All required checks have succeeded.
// * There are no merge conflicts.
func process(ctx context.Context, pr *client.PR) error {
	gaelog.Debugf(ctx, "checking if %v can be automerged", pr)

	if pr.GetMerged() || pr.GetState() != "open" {
		gaelog.Debugf(ctx, "automerge: no, not open")
		return nil
	}

	issue, err := pr.Issue(ctx)
	if err != nil {
		return err
	}

	if !issue.HasLabel(automergeLabel) {
		gaelog.Debugf(ctx, "automerge: no, does not have the %q label", automergeLabel)
		return nil
	}

	if ok, err := allReviewsFinished(ctx, pr); !ok || err != nil {
		if err != nil {
			return err
		}
		gaelog.Debugf(ctx, "automerge: no, there are unfinished reviews")
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
			gaelog.Debugf(ctx, "automerge: no, check %q missing or not successful", req)
			return nil
		}
	}

	if s := status.GetState(); s != "success" {
		gaelog.Debugf(ctx, "automerge: no, overall status is %q", s)
		return nil
	}

	ok, err := pr.Mergeable(ctx)
	if err != nil {
		return err
	}
	if !ok {
		gaelog.Debugf(ctx, "automerge: no, has merge conflicts")
		return nil
	}

	gaelog.Infof(ctx, "merging %v", pr)
	title := fmt.Sprintf("Auto-Merge pull request %v from %s/%s", pr, pr.Head.User.GetLogin(), pr.Head.GetRef())
	msg := fmt.Sprintf("Automatically merged due to %q label", automergeLabel)
	return pr.Merge(ctx, title, msg)
}

func allReviewsFinished(ctx context.Context, pr *client.PR) (bool, error) {
	reviews := map[string]string{}
	for _, u := range pr.RequestedReviewers {
		name := u.GetLogin()
		if name == "" {
			continue
		}

		reviews[name] = "REQUESTED"
	}

	revs, err := pr.Reviews(ctx)
	if err != nil {
		return false, err
	}

	for _, r := range revs {
		name := r.GetUser().GetLogin()
		state := r.GetState()
		if name == "" || state == "" {
			continue
		}

		reviews[name] = state
	}

	// To *enforce* reviews, initialize with:
	// result := len(reviews) != 0

	result := true
	for name, state := range reviews {
		if state == "APPROVED" {
			gaelog.Debugf(ctx, "Pull request %v approved by %s", pr, name)
		} else {
			gaelog.Debugf(ctx, "Review of %v by %s is in state %q", pr, name, state)
			result = false
		}
	}

	return result, nil
}
