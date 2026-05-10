package setup

import (
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
