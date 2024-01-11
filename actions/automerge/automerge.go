package automerge

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/github"
	"github.com/mtraver/gaelog"
	"github.com/octo/ghbot/client"
	"github.com/octo/ghbot/event"
)

const automergeLabel = "Automerge"

// requiredStatuses is a list of all status "contexts" that must signal success
// before a PR can automatically be merged.
// TODO(octo): may be redundant with the "Require status checks to pass before merging" setting.
var requiredStatuses = []string{
	"ChangeLog",
	"clang-format",
}

var requiredChecks = []string{
	"make_distcheck",
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

	return process(ctx, c, c.WrapPR(event.PullRequest))
}

func processReviewEvent(ctx context.Context, e *github.PullRequestReviewEvent) error {
	c, err := client.New(ctx, client.DefaultOwner, client.DefaultRepo)
	if err != nil {
		return err
	}

	return process(ctx, c, c.WrapPR(e.GetPullRequest()))
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

	return process(ctx, c, pr)
}

// process merges a pull request, if:
// * It it still open and has not already been merged.
// * Is has the Automerge label.
// * There are no outstanding reviews (CHANGES_REQUESTED or PENDING).
// * The overall state is "success".
// * All required checks have succeeded.
// * There are no merge conflicts.
func process(ctx context.Context, client *client.Client, pr *client.PR) error {
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
		gaelog.Debugf(ctx, "automerge: status %q => %q", s.GetContext(), s.GetState())
		success[s.GetContext()] = (s.GetState() == "success")
	}

	for _, req := range requiredStatuses {
		if !success[req] {
			gaelog.Debugf(ctx, "automerge: no, check %q missing or not successful", req)
			return nil
		}
	}

	if s := status.GetState(); s != "success" {
		gaelog.Debugf(ctx, "automerge: no, overall status is %q", s)
		return nil
	}

	ok, err := haveRequiredChecks(ctx, client, pr)
	if err != nil {
		return err
	}
	if !ok {
		gaelog.Debugf(ctx, "automerge: no, required checks are missing or unsuccessful")
		return nil
	}

	ok, err = pr.Mergeable(ctx)
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

func haveRequiredChecks(ctx context.Context, client *client.Client, pr *client.PR) (bool, error) {
	checkRuns, err := client.CheckRuns(ctx, pr.GetHead().GetSHA())
	if err != nil {
		return false, err
	}

	byName := make(map[string]*github.CheckRun)
	for _, cr := range checkRuns {
		gaelog.Debugf(ctx, "automerge: Check %q -> %q", cr.GetName(), cr.GetConclusion())
		byName[cr.GetName()] = cr
	}

	ret := true
	for _, name := range requiredChecks {
		cr, ok := byName[name]
		if !ok {
			gaelog.Warningf(ctx, "automerge: Required check %q was not reported by GitHub.", name)
			ret = false
		}
		if cr.GetConclusion() != "success" {
			gaelog.Debugf(ctx, "automerge: Check %q was not successful", name)
			ret = false
		}
	}

	return ret, nil
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
