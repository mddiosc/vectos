package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldExclude_SimpleWildcard(t *testing.T) {
	acc := newPathAccumulator()
	acc.projectRoot = "/project"
	acc.excludePatterns = []string{"*.log"}

	if !acc.shouldExclude("/project/app.log") {
		t.Error("*.log should match app.log")
	}
	if acc.shouldExclude("/project/app.ts") {
		t.Error("*.log should not match app.ts")
	}
}

func TestShouldExclude_DoubleStar(t *testing.T) {
	acc := newPathAccumulator()
	acc.projectRoot = "/project"
	acc.excludePatterns = []string{"src/content/**"}

	if !acc.shouldExclude("/project/src/content/blog/post.md") {
		t.Error("src/content/** should match src/content/blog/post.md")
	}
	if acc.shouldExclude("/project/lib/utils.ts") {
		t.Error("src/content/** should not match lib/utils.ts")
	}
}

func TestShouldExclude_DoubleStarNested(t *testing.T) {
	acc := newPathAccumulator()
	acc.projectRoot = "/project"
	acc.excludePatterns = []string{"**/__mocks__/**"}

	if !acc.shouldExclude("/project/pkg/__mocks__/foo.ts") {
		t.Error("**/__mocks__/** should match pkg/__mocks__/foo.ts")
	}
	if acc.shouldExclude("/project/pkg/foo.ts") {
		t.Error("**/__mocks__/** should not match pkg/foo.ts")
	}
}

func TestShouldExclude_GitignorePattern(t *testing.T) {
	acc := newPathAccumulator()
	acc.projectRoot = "/project"
	acc.excludePatterns = []string{"dist/"}

	if !acc.shouldExclude("/project/dist/bundle.js") {
		t.Error("dist/ pattern should match dist/bundle.js")
	}
}

func TestShouldExclude_EmptyPatterns(t *testing.T) {
	acc := newPathAccumulator()
	if acc.shouldExclude("/project/any/file.go") {
		t.Error("empty patterns should not match anything")
	}
}

func TestCollectIndexablePathsWithExclusions(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src", "content", "blog"), 0755)
	os.MkdirAll(filepath.Join(dir, "src", "lib"), 0755)
	os.WriteFile(filepath.Join(dir, "src", "content", "blog", "post.md"), []byte("# Post"), 0644)
	os.WriteFile(filepath.Join(dir, "src", "lib", "utils.ts"), []byte("export const x = 1;"), 0644)
	os.WriteFile(filepath.Join(dir, "debug.log"), []byte("log"), 0644)

	paths, _, err := CollectIndexablePathsWithExclusions([]string{dir}, false, []string{"src/content/**", "*.log"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, p := range paths {
		rel, _ := filepath.Rel(dir, p)
		if rel == filepath.Join("src", "content", "blog", "post.md") {
			t.Error("blog/post.md should have been excluded by src/content/**")
		}
		if rel == "debug.log" {
			t.Error("debug.log should have been excluded by *.log")
		}
	}

	found := false
	for _, p := range paths {
		rel, _ := filepath.Rel(dir, p)
		if rel == filepath.Join("src", "lib", "utils.ts") {
			found = true
		}
	}
	if !found {
		t.Error("src/lib/utils.ts should NOT have been excluded")
	}
}