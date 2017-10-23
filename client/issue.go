package client

import (
	"fmt"

	"github.com/google/go-github/github"
)

type Issue struct {
	client *Client
	*github.Issue
}

func (i *Issue) Number() int {
	if i == nil || i.Issue == nil || i.Issue.Number == nil {
		return -1
	}
	return *i.Issue.Number
}

func (i *Issue) String() string {
	return fmt.Sprintf("#%d", i.Number())
}

func (i *Issue) HasLabel(name string) bool {
	for _, label := range i.Issue.Labels {
		if label.Name != nil && *label.Name == name {
			return true
		}
	}

	return false
}
