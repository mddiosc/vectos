package usererr

import (
	"fmt"
	"os"
)

// WrapPathOp turns low-level filesystem errors into clearer user-facing errors.
func WrapPathOp(action, noun, path string, err error) error {
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		return fmt.Errorf("cannot %s %s %s: it does not exist", action, noun, path)
	}
	if os.IsPermission(err) {
		return fmt.Errorf("cannot %s %s %s: permission denied", action, noun, path)
	}
	return fmt.Errorf("cannot %s %s %s: %w", action, noun, path, err)
}
