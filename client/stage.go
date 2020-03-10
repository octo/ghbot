package client

import (
	"context"
	"sync"

	"github.com/google/go-github/github"
)

type Stage struct {
	git *github.GitService
	mu  *sync.Mutex

	owner  string
	repo   string
	ref    string
	commit string

	entries []*github.TreeEntry
}

func (c *Client) NewStage(pr *github.PullRequest) *Stage {
	return &Stage{
		git:    c.Git,
		mu:     &sync.Mutex{},
		owner:  pr.Head.Repo.Owner.GetLogin(),
		repo:   pr.Head.Repo.GetName(),
		ref:    pr.Head.GetRef(),
		commit: pr.Head.GetSHA(),
	}
}

func (s *Stage) Add(path, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append(s.entries, &github.TreeEntry{
		Path:    github.String(path),
		Mode:    github.String("100644"),
		Type:    github.String("blob"),
		Content: github.String(content),
	})
}

func (s *Stage) Commit(ctx context.Context, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.entries == nil {
		return nil
	}

	baseCommit, _, err := s.git.GetCommit(ctx, s.owner, s.repo, s.commit)
	if err != nil {
		return err
	}

	commitTree, _, err := s.git.CreateTree(ctx, s.owner, s.repo, baseCommit.Tree.GetSHA(), s.entries)
	if err != nil {
		return err
	}

	commit, _, err := s.git.CreateCommit(ctx, s.owner, s.repo, &github.Commit{
		Message: github.String(message),
		Tree:    commitTree,
		Parents: []*github.Commit{baseCommit},
	})
	if err != nil {
		return err
	}

	_, _, err = s.git.UpdateRef(ctx, s.owner, s.repo, &github.Reference{
		Ref: github.String(s.ref),
		Object: &github.GitObject{
			Type: github.String("commit"),
			SHA:  commit.SHA,
		},
	}, false)
	if err != nil {
		return err
	}

	s.entries = nil
	return nil
}
