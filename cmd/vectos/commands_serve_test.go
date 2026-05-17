package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestComputeFileHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	hash, err := computeFileHash(path)
	if err != nil {
		t.Fatal(err)
	}
	expected := sha256.Sum256([]byte("hello world"))
	if hash != hex.EncodeToString(expected[:]) {
		t.Fatalf("hash = %s, want %s", hash, hex.EncodeToString(expected[:]))
	}
	if err := os.WriteFile(path, []byte("hello vectos"), 0644); err != nil {
		t.Fatal(err)
	}
	hash2, err := computeFileHash(path)
	if err != nil {
		t.Fatal(err)
	}
	if hash2 == hash {
		t.Fatal("expected hash to change after file modification")
	}
	emptyPath := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(emptyPath, nil, 0644); err != nil {
		t.Fatal(err)
	}
	emptyHash, err := computeFileHash(emptyPath)
	if err != nil {
		t.Fatal(err)
	}
	emptyExpected := sha256.Sum256(nil)
	if emptyHash != hex.EncodeToString(emptyExpected[:]) {
		t.Fatalf("empty hash = %s, want %s", emptyHash, hex.EncodeToString(emptyExpected[:]))
	}
}

func TestComputeFileHashWrapsReadErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.txt")
	_, err := computeFileHash(missing)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "cannot read file "+missing+": it does not exist" {
		t.Fatalf("unexpected error: %v", err)
	}
}
