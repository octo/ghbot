package client

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/go-github/github"
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

// Mergeable returns true if the pull request can be merged without conflicts.
func (pr *PR) Mergeable(ctx context.Context) (bool, error) {
	if pr.PullRequest.Mergeable == nil {
		// the "Mergeable" field is not always populated, e.g. by the
		// List() call, so retrieve the PR information again …
		fullPR, _, err := pr.client.PullRequests.Get(ctx, pr.client.owner, pr.client.repo, pr.Number())
		if err != nil {
			return false, err
		}
		pr.PullRequest = fullPR

		if pr.PullRequest.Mergeable == nil {
			log.Printf(`PR %v: unable to determine state of the "Mergeable" flag`, pr)
			return false, errors.New(`unable to determine state of the "Mergeable" flag`)
		}
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

	if res.Merged == nil || !*res.Merged {
		log.Printf("did not merge %v: %s", pr, *res.Message)
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

	return b.GetContent(), nil
}
