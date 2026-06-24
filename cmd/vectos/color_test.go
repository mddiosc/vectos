package main

import (
	"strings"
	"testing"
)

func TestColorizeDisabledIsPlain(t *testing.T) {
	old := colorEnabled
	colorEnabled = false
	defer func() { colorEnabled = old }()

	if got := bold("hi"); got != "hi" {
		t.Errorf("disabled colorize should return plain text, got %q", got)
	}
}

func TestColorizeEnabledWrapsWithReset(t *testing.T) {
	old := colorEnabled
	colorEnabled = true
	defer func() { colorEnabled = old }()

	got := green("ok")
	if !strings.HasPrefix(got, ansiGreen) || !strings.HasSuffix(got, ansiReset) {
		t.Errorf("enabled colorize should wrap with code+reset, got %q", got)
	}
	// empty string stays empty even when enabled
	if green("") != "" {
		t.Error("empty input should stay empty")
	}
}

func TestColorScoreThresholds(t *testing.T) {
	old := colorEnabled
	colorEnabled = true
	defer func() { colorEnabled = old }()

	cases := []struct {
		score float64
		code  string
	}{
		{0.7, ansiGreen},
		{0.5, ansiYellow},
		{0.1, ansiDim},
	}
	for _, c := range cases {
		if got := colorScore(c.score); !strings.HasPrefix(got, c.code) {
			t.Errorf("colorScore(%v) = %q, want prefix %q", c.score, got, c.code)
		}
	}
}
