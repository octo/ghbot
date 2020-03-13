#!/usr/bin/perl

use strict;
use warnings;

# Not supported:
#
#   Download
#   Follow
#   ForkApply
#   Gist

my @eventTypes = qw(
    CheckRun
    CheckSuite
    CommitComment
    Create
    Delete
    DeployKey
    Deployment
    DeploymentStatus
    Fork
    GitHubAppAuthorization
    Gollum
    Installation
    InstallationRepositories
    IssueComment
    Issue
    Issues
    Label
    MarketplacePurchase
    Member
    Membership
    Meta
    Milestone
    Organization
    OrgBlock
    PageBuild
    ProjectCard
    ProjectColumn
    Project
    Public
    PullRequest
    PullRequestReview
    PullRequestReviewComment
    Push
    Release
    RepositoryDispatch
    Repository
    RepositoryVulnerabilityAlert
    Star
    Status
    TeamAdd
    Team
    User
    Watch
);

print <<EOF;
// This file was generated by $0

package event // import "github.com/octo/ghbot/event"

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/google/go-github/github"
	"go.opencensus.io/trace"
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
var $global_var = map[string]func(context.Context, *github.${type}Event) error{}

// ${type}Handler registers a handler for ${type} events.
func ${type}Handler(name string, hndl func(context.Context, *github.${type}Event) error) {
	$global_var\[name\] = hndl
}

// handle${type} calls all handlers for ${type} events. If a handler
// returns an error, that error is returned immediately and no further handlers
// are called.
func handle${type}(ctx context.Context, event *github.${type}Event) error {
	ctx, span := trace.StartSpan(ctx, "Event ${type}")
	span.AddAttributes(
		trace.StringAttribute("/github/event", "${type}"),
	)
	defer span.End()

	wg := sync.WaitGroup{}
	ch := make(chan error)

	for name, hndl := range $global_var {
		wg.Add(1)

		go func(name string, hndl func(context.Context, *github.${type}Event) error) {
			defer wg.Done()

			ctx, span := trace.StartSpan(ctx, "Action "+name)
			span.AddAttributes(
				trace.StringAttribute("/github/bot/action", name),
			)
			defer span.End()

			if err := hndl(ctx, event); err != nil {
				ch <- fmt.Errorf("%q ${type} handler: %v", name, err)
			}
		}(name, hndl)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var lastErr error
	for err := range ch {
		if lastErr != nil {
			log.Print(lastErr)
		}
		lastErr = err
	}

	return lastErr
}
EOF
}
