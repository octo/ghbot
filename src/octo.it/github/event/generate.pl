#!/usr/bin/perl

use strict;
use warnings;

# Not supported:
#
#   Download
#   Follow
#   ForkApply
#   Gist
#   Organization
#   Team

my @eventTypes = qw(
    CommitComment
    Create
    Delete
    Deployment
    DeploymentStatus
    Fork
    Gollum
    IssueComment
    Issues
    Label
    Member
    Membership
    Milestone
    PageBuild
    Public
    PullRequest
    PullRequestReview
    PullRequestReviewComment
    Push
    Release
    Repository
    Status
    TeamAdd
    Watch
);

print <<EOF;
// This file was generated by $0

package event // import "octo.it/github/event"

import (
	"context"
	"log"

	"github.com/google/go-github/github"
)

// Handle handles a webhook event.
func Handle(ctx context.Context, event interface{}) error {
	switch event := event.(type) {
EOF
for (@eventTypes) {
	my $type = $_;
	print <<EOF;
	case *github.${type}Event:
		return handle${type}(ctx, event)
EOF
}
print <<'EOF';
	default:
		log.Printf("unimplemented event type: %T", event)
	}

	return nil
}
EOF

for (@eventTypes) {
	my $type = $_;
	my $global_var = lcfirst($type) . 'Handlers';

	print <<EOF;

//
// $type events
//
var $global_var []func(context.Context, *github.${type}Event) error

// ${type}Handler registers a handler for ${type} events.
func ${type}Handler(hndl func(context.Context, *github.${type}Event) error) {
	$global_var = append($global_var, hndl)
}

// handle${type} calls all handlers for ${type} events. If a handler
// returns an error, that error is returned immediately and no further handlers
// are called.
func handle${type}(ctx context.Context, event *github.${type}Event) error {
	for _, hndl := range $global_var {
		if err := hndl(ctx, event); err != nil {
			return err
		}
	}

	return nil
}
EOF
}
