package main

import (
	"errors"
	"fmt"

	"github.com/dhitalkamal/parley/internal/platform/fsstore"
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
)

// tightenWorkspaces sweeps every registered workspace's ProjectRoot (once per
// distinct root) so secret files written by older builds at 0644 get
// restricted. Callers treat a failure as a warning, never as fatal.
func tightenWorkspaces(workspaces []workspace.Workspace) error {
	seen := map[string]bool{}
	var errs []error
	for _, w := range workspaces {
		if w.ProjectRoot == "" || seen[w.ProjectRoot] {
			continue
		}
		seen[w.ProjectRoot] = true
		if err := fsstore.TightenSecretFiles(w.ProjectRoot); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// permissionWarning is the one-line summary shown for a failed sweep, or ""
// when there was nothing to report.
func permissionWarning(err error) string {
	if err == nil {
		return ""
	}
	first := err
	for {
		multi, ok := first.(interface{ Unwrap() []error })
		if !ok || len(multi.Unwrap()) == 0 {
			break
		}
		first = multi.Unwrap()[0]
	}
	return fmt.Sprintf("parley: warning: could not restrict permissions on %d file(s): %v", countErrors(err), first)
}

// countErrors counts the leaves of nested errors.Join results.
func countErrors(err error) int {
	if err == nil {
		return 0
	}
	multi, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return 1
	}
	n := 0
	for _, e := range multi.Unwrap() {
		n += countErrors(e)
	}
	return n
}
