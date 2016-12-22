package client

import (
	"context"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	DefaultOwner = "collectd"
	DefaultRepo  = "collectd"
)

var accessToken = "@SECRET@"

type Client struct {
	owner string
	repo  string

	*github.Client
}

func New(ctx context.Context, owner, repo string) *Client {
	return &Client{
		owner: owner,
		repo:  repo,
		Client: github.NewClient(oauth2.NewClient(ctx,
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken}))),
	}
}

func (c *Client) PullRequestBySHA(sha string) (*github.PullRequest, error) {
	opts := github.PullRequestListOptions{}

	for {
		prs, res, err := c.PullRequests.List(c.owner, c.repo, &opts)
		if err != nil {
			return nil, err
		}

		for _, pr := range prs {
			if *pr.Head.SHA == sha {
				return pr, nil
			}
		}

		if res.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = res.NextPage
	}

	return nil, os.ErrNotExist
}
