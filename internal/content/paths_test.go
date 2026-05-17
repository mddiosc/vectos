package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldExclude_SimpleWildcard(t *testing.T) {
	acc := newPathAccumulator()
	acc.projectRoot = "/project"
	acc.excludePatterns = compileExcludePatterns([]string{"*.log"})

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
	acc.excludePatterns = compileExcludePatterns([]string{"src/content/**"})

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
	acc.excludePatterns = compileExcludePatterns([]string{"**/__mocks__/**"})

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
	acc.excludePatterns = compileExcludePatterns([]string{"dist/"})

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

func TestCompileExcludePatterns_DedupesAndNormalizes(t *testing.T) {
	got := compileExcludePatterns([]string{
		"dist/",
		"dist/",          // exact duplicate
		"  dist/  ",      // whitespace duplicate after trim
		"dist/**",        // duplicate after normalization of "dist/"
		"",               // empty -> dropped
		"   ",            // whitespace-only -> dropped
		"src/content/**", // unique
	})
	if len(got) != 2 {
		t.Fatalf("expected 2 unique compiled patterns, got %d: %#v", len(got), got)
	}
	seen := map[string]bool{}
	for _, c := range got {
		seen[c.pattern] = true
	}
	if !seen["dist/**"] || !seen["src/content/**"] {
		t.Errorf("expected dist/** and src/content/** to remain, got %#v", got)
	}
}

func TestShouldExcludeDir_BareDirectoryPattern(t *testing.T) {
	acc := newPathAccumulator()
	acc.projectRoot = "/project"
	acc.excludePatterns = compileExcludePatterns([]string{"node_modules"})

	if !acc.shouldExcludeDir("/project/node_modules") {
		t.Error("bare 'node_modules' pattern should match the directory itself")
	}
	if !acc.shouldExcludeDir("/project/pkg/node_modules") {
		t.Error("bare 'node_modules' should match nested node_modules dir by base name")
	}
	if acc.shouldExcludeDir("/project/src") {
		t.Error("'node_modules' should not match other directories")
	}
}

func TestShouldExcludeDir_TrailingSlashPattern(t *testing.T) {
	acc := newPathAccumulator()
	acc.projectRoot = "/project"
	acc.excludePatterns = compileExcludePatterns([]string{"dist/"})

	if !acc.shouldExcludeDir("/project/dist") {
		t.Error("'dist/' should match the dist directory so walkDir can prune it")
	}
}

func TestCollectIndexablePathsWithExclusions(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src", "content", "blog"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src", "lib"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "content", "blog", "post.md"), []byte("# Post"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "lib", "utils.ts"), []byte("export const x = 1;"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "debug.log"), []byte("log"), 0644); err != nil {
		t.Fatal(err)
	}

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

// TestCollectIndexablePathsWithExclusions_PrunesExcludedDirs verifies that
// when a directory matches an exclusion pattern, walkDir does not descend
// into it. We assert this indirectly: a non-indexable file deep inside the
// excluded subtree must NOT appear in the skipped list either — because
// SkipDir prevents the file from being visited at all.
func TestCollectIndexablePathsWithExclusions_PrunesExcludedDirs(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	deepFile := filepath.Join(dir, "node_modules", "pkg", "index.js")
	if err := os.WriteFile(deepFile, []byte("module.exports = {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "utils.ts"), []byte("export const x = 1;"), 0644); err != nil {
		t.Fatal(err)
	}

	paths, skipped, err := CollectIndexablePathsWithExclusions([]string{dir}, false, []string{"node_modules/"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, p := range paths {
		if p == deepFile {
			t.Errorf("file under excluded node_modules/ should not be indexed: %s", p)
		}
	}
	// The whole subtree should be pruned by SkipDir, so the deep file
	// should not show up in `skipped` either.
	for _, p := range skipped {
		if p == deepFile {
			t.Errorf("file under excluded node_modules/ should be pruned (not even reported as skipped): %s", p)
		}
	}
	if len(paths) == 0 {
		t.Fatal("expected at least src/utils.ts to be indexed")
	}
}

func TestCollectIndexablePathsWithExclusions_WrapsMissingRootError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	_, _, err := CollectIndexablePathsWithExclusions([]string{missing}, false, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "cannot access path "+missing+": it does not exist" {
		t.Fatalf("unexpected error: %v", err)
	}
}
