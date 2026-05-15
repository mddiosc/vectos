package storage

import "testing"

func TestIsDocFilePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"docs prefix unix", "docs/en/DEVELOPMENT.md", true},
		{"docs subdir unix", "project/docs/api.md", true},
		{"docs prefix windows", `project\docs\guide.md`, true},
		{"readme.md root", "README.md", true},
		{"readme.md case insensitive", "Readme.Md", true},
		{"readme.md nested", "packages/lib/README.md", true},
		{"blog post", "src/content/blog/en/post.md", false},
		{"regular source", "src/auth/middleware.ts", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDocFilePath(tt.path); got != tt.want {
				t.Errorf("isDocFilePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

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
