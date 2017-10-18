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

func (c *Client) Issue(ctx context.Context, number int) (*Issue, error) {
	issue, _, err := c.Client.Issues.Get(ctx, c.owner, c.repo, number)
	if err != nil {
		return nil, err
	}

	return &Issue{
		client: c,
		Issue:  issue,
	}, nil
}

func (c *Client) WrapIssue(issue *github.Issue) *Issue {
	return &Issue{
		client: c,
		Issue:  issue,
	}
}

func (c *Client) PullRequestBySHA(ctx context.Context, sha string) (*PR, error) {
	opts := github.PullRequestListOptions{}

	for {
		prs, res, err := c.PullRequests.List(ctx, c.owner, c.repo, &opts)
		if err != nil {
			return nil, err
		}

		for _, pr := range prs {
			if *pr.Head.SHA == sha {
				return c.WrapPR(pr), nil
			}
		}

		if res.NextPage == 0 {
			break
		}
		opts.ListOptions.Page = res.NextPage
	}

	return nil, os.ErrNotExist
}

func (c *Client) WrapPR(pr *github.PullRequest) *PR {
	return &PR{
		client:      c,
		PullRequest: pr,
	}
}
