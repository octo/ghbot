package client

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"

	"github.com/google/go-github/github"
	"github.com/octo/retry"
)

type PR struct {
	client *Client
	*github.PullRequest
}

func (pr *PR) Number() int {
	if pr == nil || pr.PullRequest == nil || pr.PullRequest.Number == nil {
		return -1
	}
	return *pr.PullRequest.Number
}

func (pr *PR) String() string {
	return fmt.Sprintf("#%d", pr.Number())
}

// fetchMergeable gets the pull request from the API and returns an error if
// the "mergeable" field is not set.
func (pr *PR) fetchMergeable(ctx context.Context) error {
	fullPR, _, err := pr.client.PullRequests.Get(ctx, pr.client.owner, pr.client.repo, pr.Number())
	if err != nil {
		return err
	}
	pr.PullRequest = fullPR

	if pr.PullRequest.Mergeable == nil {
		return errors.New(`unable to determine state of the "Mergeable" flag`)
	}

	return nil
}

// Mergeable returns true if the pull request can be merged without conflicts.
func (pr *PR) Mergeable(ctx context.Context) (bool, error) {
	if pr.PullRequest.Mergeable != nil {
		return *pr.PullRequest.Mergeable, nil
	}

	if err := retry.Do(ctx, pr.fetchMergeable); err != nil {
		return false, err
	}

	return *pr.PullRequest.Mergeable, nil
}

// Merge merges the pull request.
func (pr *PR) Merge(ctx context.Context, title, msg string) error {
	opts := &github.PullRequestOptions{
		CommitTitle: title,
		MergeMethod: "merge",
	}

	res, _, err := pr.client.PullRequests.Merge(ctx, pr.client.owner, pr.client.repo, pr.Number(), msg, opts)
	if err != nil {
		return err
	}

	if !res.GetMerged() {
		log.Printf("did not merge %v: %s", pr, res.GetMessage())
		return nil
	}

	return nil
}

// CombinedStatus ...
func (pr *PR) CombinedStatus(ctx context.Context) (*github.CombinedStatus, error) {
	status, _, err := pr.client.Repositories.GetCombinedStatus(ctx, pr.client.owner, pr.client.repo, *pr.Head.SHA, &github.ListOptions{})
	return status, err
}

func (pr *PR) Issue(ctx context.Context) (*Issue, error) {
	return pr.client.Issue(ctx, pr.Number())
}

type PRFile struct {
	Filename string
	SHA      string
}

// Files returns the files that are added or modified by this PR. Files without
// SHA (deleted files) are not returned.
func (pr *PR) Files(ctx context.Context) ([]PRFile, error) {
	opts := &github.ListOptions{}

	var ret []PRFile
	for {
		files, res, err := pr.client.PullRequests.ListFiles(ctx, pr.client.owner, pr.client.repo, pr.Number(), opts)
		if err != nil {
			return nil, err
		}

		for _, f := range files {
			if f.SHA == nil {
				continue
			}

			ret = append(ret, PRFile{
				Filename: f.GetFilename(),
				SHA:      f.GetSHA(),
			})
		}

		if res.NextPage == 0 {
			break
		}

		opts.Page = res.NextPage
	}

	return ret, nil
}

func (pr *PR) Blob(ctx context.Context, sha string) (string, error) {
	repo := pr.PullRequest.Head.Repo

	b, _, err := pr.client.Git.GetBlob(ctx, repo.Owner.GetLogin(), repo.GetName(), sha)
	if err != nil {
		return "", err
	}

	c, err := base64.StdEncoding.DecodeString(b.GetContent())
	if err != nil {
		return "", err
	}

	return string(c), nil
}

func (pr *PR) Reviews(ctx context.Context) ([]*github.PullRequestReview, error) {
	var (
		opts = &github.ListOptions{}
		ret  []*github.PullRequestReview
	)

	for {
		revs, res, err := pr.client.PullRequests.ListReviews(ctx, pr.client.owner, pr.client.repo, pr.Number(), opts)
		if err != nil {
			return nil, fmt.Errorf("PullRequests.ListReviews(%v): %v", pr, err)
		}

		ret = append(ret, revs...)

		if res.NextPage == 0 {
			break
		}
		opts.Page = res.NextPage
	}

	return ret, nil
}
