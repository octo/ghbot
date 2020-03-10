package client

import (
	"testing"
)

func TestTrimLength(t *testing.T) {
	//             123456789012
	const input = "collectd ♥♥♥"

	cases := []struct {
		s      string
		length uint
		want   string
	}{
		{"collectd ♥♥♥", 15, "collectd ♥♥♥"},
		{"collectd ♥♥♥", 12, "collectd ♥♥♥"},
		{"collectd ♥♥♥", 11, "collectd ♥…"},
		{"collectd ♥♥♥", 10, "collectd …"},
		{"collectd ♥♥♥", 9, "collectd…"},
		{"foo", 1, "…"},
		{"foo", 0, ""},
	}

	for _, tc := range cases {
		got := trimLength(tc.s, tc.length)

		if got != tc.want {
			t.Errorf("trimLength(%q, %d) = %q, want %q", tc.s, tc.length, got, tc.want)
		}
	}
}
