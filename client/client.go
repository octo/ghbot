package client

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/octo/ghbot/config"
	"github.com/octo/retry"
	"golang.org/x/oauth2"
)

const (
	DefaultOwner = "collectd"
	DefaultRepo  = "collectd"

	StatusSuccess = "success"
	StatusFailure = "failure"
	StatusError   = "error"
	StatusPending = "pending"
)

type Client struct {
	owner string
	repo  string

	*github.Client
}

func New(ctx context.Context, owner, repo string) (*Client, error) {
	accessToken, err := config.AccessToken(ctx)
	if err != nil {
		return nil, err
	}

	t := &retry.Transport{
		RoundTripper: &oauth2.Transport{
			Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken}),
			Base:   http.DefaultTransport,
		},
	}

	return &Client{
		owner: owner,
		repo:  repo,
		Client: github.NewClient(&http.Client{
			Transport: t,
		}),
	}, nil
}

func (c *Client) Issue(ctx context.Context, number int) (*Issue, error) {
	issue, _, err := c.Client.Issues.Get(ctx, c.owner, c.repo, number)
	if err != nil {
		return nil, err
	}

	return c.WrapIssue(issue), nil
}

func (c *Client) WrapIssue(issue *github.Issue) *Issue {
	return &Issue{
		client: c,
		Issue:  issue,
	}
}

func (c *Client) PullRequestBySHA(ctx context.Context, sha string) (*PR, error) {
	refs, _, err := c.Git.GetRefs(ctx, c.owner, c.repo, "pull/")
	if err != nil {
		return nil, fmt.Errorf("Git.GetRefs(): %v", err)
	}

	for _, ref := range refs {
		if ref.Object.GetSHA() != sha {
			continue
		}

		m := regexp.MustCompile("^refs/pull/([1-9][0-9]*)/").FindStringSubmatch(ref.GetRef())
		if m == nil {
			// no match
			continue
		}

		number, err := strconv.Atoi(m[1])
		if err != nil {
			log.Printf("strconv.Atoi(%q): %v", m[1], err)
			continue
		}

		return c.PR(ctx, number)
	}

	return nil, os.ErrNotExist
}

func (c *Client) PR(ctx context.Context, number int) (*PR, error) {
	pr, _, err := c.PullRequests.Get(ctx, c.owner, c.repo, number)
	if err != nil {
		return nil, fmt.Errorf("PullRequests.Get(%d): %v", number, err)
	}

	return c.WrapPR(pr), nil
}

func (c *Client) WrapPR(pr *github.PullRequest) *PR {
	return &PR{
		client:      c,
		PullRequest: pr,
	}
}

func (c *Client) CreateStatus(ctx context.Context, name, state, desc, url, ref string) error {
	req := &github.RepoStatus{
		State:       &state,
		Description: &desc,
		Context:     &name,
	}
	if url != "" {
		req.TargetURL = &url
	}

	_, _, err := c.Repositories.CreateStatus(ctx, c.owner, c.repo, ref, req)
	return err
}

func (c *Client) Milestones(ctx context.Context) (map[string]int, error) {
	var (
		ret  = make(map[string]int)
		opts github.MilestoneListOptions
	)

	for {
		ms, res, err := c.Issues.ListMilestones(ctx, c.owner, c.repo, &opts)
		if err != nil {
			return nil, err
		}

		for _, m := range ms {
			ret[m.GetTitle()] = m.GetNumber()
		}

		if res.NextPage == 0 {
			break
		}
		opts.Page = res.NextPage
	}

	return ret, nil
}

func (c *Client) FormatUser(ctx context.Context, login string) string {
	u, _, err := c.Users.Get(ctx, login)
	if err != nil || u.GetName() == "" {
		return "@" + login
	}

	return fmt.Sprintf("@%s (%s)", login, u.GetName())
}
