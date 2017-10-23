package format

import (
	"testing"
)

func TestHasAnySuffix(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"test.c", true},
		{"test.h", true},
		{"test.proto", true},
		{"test.db", false},
		{"types.db", false},
	}

	for _, c := range cases {
		got := hasAnySuffix(c.in, []string{".c", ".h", ".proto"})
		if got != c.want {
			t.Errorf("hasAnySuffix(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
