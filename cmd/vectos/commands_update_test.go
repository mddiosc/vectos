package main

import (
	"strings"
	"testing"
)

func TestDecodeRelease(t *testing.T) {
	body := `{"tag_name":"v1.5.0","name":"v1.5.0","body":"- fix x\n- add y","html_url":"https://example/r"}`
	rel, err := decodeRelease(strings.NewReader(body))
	if err != nil {
		t.Fatalf("decodeRelease: %v", err)
	}
	if rel.TagName != "v1.5.0" {
		t.Errorf("TagName = %q, want v1.5.0", rel.TagName)
	}
	if !strings.Contains(rel.Body, "add y") {
		t.Errorf("Body missing notes: %q", rel.Body)
	}
}

func TestDecodeReleaseNoTag(t *testing.T) {
	if _, err := decodeRelease(strings.NewReader(`{"name":"x"}`)); err == nil {
		t.Error("expected error when tag_name is empty")
	}
}
