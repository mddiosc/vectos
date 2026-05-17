package usererr

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestWrapPathOp_NotExist(t *testing.T) {
	err := WrapPathOp("read", "file", "/tmp/missing.txt", os.ErrNotExist)
	if err == nil || !strings.Contains(err.Error(), "cannot read file /tmp/missing.txt: it does not exist") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWrapPathOp_Permission(t *testing.T) {
	err := WrapPathOp("read", "file", "/tmp/secret.txt", os.ErrPermission)
	if err == nil || !strings.Contains(err.Error(), "cannot read file /tmp/secret.txt: permission denied") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWrapPathOp_PreservesUnknownErrors(t *testing.T) {
	err := WrapPathOp("read", "file", "/tmp/data.txt", errors.New("i/o failure"))
	if err == nil || !strings.Contains(err.Error(), "i/o failure") {
		t.Fatalf("unexpected error: %v", err)
	}
}
