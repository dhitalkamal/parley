// Package fsstore holds filesystem-store helpers shared across the aggregate
// stores: request name validation and the on-disk request JSON shape. Keeping
// them here stops the environment/workspace/history stores depending on the
// collection store just to reuse them.
package fsstore

import (
	"errors"
	"strings"
)

// ValidateName rejects names that would escape a store root or contain path
// separators, so a display name can round-trip through an on-disk file name.
func ValidateName(name string) error {
	if name == "" {
		return errors.New("name must not be empty")
	}
	if name == "." || name == ".." {
		return errors.New("name must not be \".\" or \"..\"")
	}
	if strings.ContainsAny(name, "/\\") {
		return errors.New("name must not contain path separators")
	}
	return nil
}
