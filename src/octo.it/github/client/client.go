package client

import (
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/appengine/urlfetch"
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
		Client: github.NewClient(&http.Client{
			Transport: &oauth2.Transport{
				Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken}),
				Base: &urlfetch.Transport{
					Context: ctx,
				},
			},
		}),
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
