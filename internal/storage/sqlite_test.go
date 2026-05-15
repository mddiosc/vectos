package storage

import "testing"

func TestEscapeLikeTerm(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal term", "auth middleware", "auth middleware"},
		{"percent sign", "100% coverage", `100\% coverage`},
		{"underscore", "my_func", `my\_func`},
		{"backslash", `path\to\file`, `path\\to\\file`},
		{"all wildcards", `%test_100%`, `\%test\_100\%`},
		{"multiple percent", "%%%%", `\%\%\%\%`},
		{"mixed content", `test\\case_100%`, `test\\\\case\_100\%`},
		{"empty string", "", ""},
		{"no special chars", "hello world", "hello world"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeLikeTerm(tt.input)
			if got != tt.want {
				t.Errorf("escapeLikeTerm(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
