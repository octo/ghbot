package changelog

import (
	"testing"
)

func TestRegexp(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"Summary\n", false},
		{"Summary\nChangeLog=some string", true},
		{"Summary\nChangeLog = some string", true},
		{"Summary\nchangelog=some string", true},
		{"Summary\nchangelog = some string", true},
		{"Summary\nChange Log=some string", false},
		{"Summary\nChange Log = some string", false},
	}

	for _, tc := range cases {
		got := logEntryRE.MatchString(tc.in)
		if got != tc.want {
			t.Errorf("logEntryRE.MatchString(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
