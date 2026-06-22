package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagedGuidanceMentionsDocsSearchAndDocsIndexing(t *testing.T) {
	guidance := managedGuidance("<!-- start -->", "<!-- end -->")

	checks := []string{
		"vectos_search_code",
		"search_code",
		"vectos_search_docs",
		"search_docs",
		"vectos_index_project",
		"index_project",
		"docs: true",
		// Consolidated incremental reindex mention
		"incremental refresh",
		// Nx lib coverage wording
		"VECTOS_NX_INCLUDE_E2E",
		"internal dependency libs",
		// Delegation guidance
		"specialist agents that perform code search",
		"enforcement point",
	}

	for _, want := range checks {
		if !strings.Contains(guidance, want) {
			t.Fatalf("expected guidance to contain %q", want)
		}
	}
}

func TestManagedGuidanceUsesProvidedMarkers(t *testing.T) {
	guidance := managedGuidance("<!-- alpha -->", "<!-- omega -->")
	if !strings.HasPrefix(guidance, "<!-- alpha -->") {
		t.Fatalf("expected custom start marker, got %q", guidance)
	}
	if !strings.HasSuffix(guidance, "<!-- omega -->") {
		t.Fatalf("expected custom end marker, got %q", guidance)
	}
}

func TestOpenCodePluginGuidanceMatchesManagedGuidance(t *testing.T) {
	pluginPath := filepath.Join("plugins", "vectos.ts")
	plugin, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatalf("read %s: %v", pluginPath, err)
	}

	got, ok := extractTSBacktickConst(string(plugin), "VECTOS_GUIDANCE")
	if !ok {
		t.Fatal("expected VECTOS_GUIDANCE backtick const in plugin")
	}

	const start = "<!-- start -->"
	const end = "<!-- end -->"
	want := managedGuidance(start, end)
	want = strings.TrimPrefix(want, start+"\n")
	want = strings.TrimSuffix(want, "\n"+end)

	if got != want {
		t.Fatalf("VECTOS_GUIDANCE drifted from managedGuidance\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func extractTSBacktickConst(src string, name string) (string, bool) {
	marker := "const " + name + " = `"
	start := strings.Index(src, marker)
	if start < 0 {
		return "", false
	}

	var out strings.Builder
	for i := start + len(marker); i < len(src); i++ {
		ch := src[i]
		if ch == '`' {
			return out.String(), true
		}
		if ch == '\\' && i+1 < len(src) {
			next := src[i+1]
			if next == '`' || next == '\\' {
				out.WriteByte(next)
				i++
				continue
			}
		}
		out.WriteByte(ch)
	}

	return "", false
}
