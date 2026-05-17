package usererr

import (
	"fmt"
	"os"
)

type wrappedUserError struct {
	message string
	cause   error
}

func (e *wrappedUserError) Error() string {
	return e.message
}

func (e *wrappedUserError) Unwrap() error {
	return e.cause
}

// WrapPathOp turns low-level filesystem errors into clearer user-facing errors.
func WrapPathOp(action, noun, path string, err error) error {
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		return &wrappedUserError{message: fmt.Sprintf("cannot %s %s %s: it does not exist", action, noun, path), cause: err}
	}
	if os.IsPermission(err) {
		return &wrappedUserError{message: fmt.Sprintf("cannot %s %s %s: permission denied", action, noun, path), cause: err}
	}
	return fmt.Errorf("cannot %s %s %s: %w", action, noun, path, err)
}
