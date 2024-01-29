package labels

import (
	"testing"

	"github.com/google/go-github/github"
	"github.com/octo/ghbot/client"
)

func TestGuessLabel(t *testing.T) {
	cases := []struct {
		title string
		want  string
	}{
		{"feat: Something new", labelFeature},
		{"feat(something): New", labelFeature},
		{"fix: Something broken", labelFix},
		{"[collectd 6] feat: v6 feature", labelFeature},
		{"feat: [collectd 6] v6 feature", labelFeature},
		{"fix: [collectd 6] feat: v6 feature", labelFix},
		{"chore: Maintenance", labelMaintenance},
		{"perf: Performance is a feature", labelFeature},
		{"docs: Documentation is a feature", labelFix},
		{"testing: unknown label", ""},
	}

	for _, tc := range cases {
		wantOK := true
		if tc.want == "" {
			wantOK = false
		}

		pr := &client.PR{
			PullRequest: &github.PullRequest{
				Title: &tc.title,
			},
		}

		got, gotOK := guessLabel(pr)
		if got != tc.want || gotOK != wantOK {
			t.Errorf("guessLabel(%q) = (%q, %v), want (%q, %v)", tc.title, got, gotOK, tc.want, wantOK)
		}
	}
}
