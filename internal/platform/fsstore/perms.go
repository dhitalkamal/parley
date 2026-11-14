package fsstore

import (
	"errors"
	"os"
	"path/filepath"
)

// WriteOwnerOnly writes data to path with mode 0600. os.WriteFile applies its
// perm only when it creates the file, so a file left at 0644 by an older
// build would stay readable; the explicit Chmod tightens it.
func WriteOwnerOnly(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// TightenSecretFiles chmods the secret-bearing files under one workspace
// ProjectRoot to 0600 if they are group or world accessible. collections/ and
// workspaces.json are deliberately not listed (collections are shared through
// git). Missing files are skipped, symlinks are never followed, and a failure
// on one file does not stop the rest; failures come back joined.
func TightenSecretFiles(projectRoot string) error {
	paths := []string{
		filepath.Join(projectRoot, "globals.json"),
		filepath.Join(projectRoot, "active_environment"),
		filepath.Join(projectRoot, "history.jsonl"),
		filepath.Join(projectRoot, "last_responses.json"),
		filepath.Join(projectRoot, "runs.jsonl"),
	}
	var errs []error
	envs, err := filepath.Glob(filepath.Join(projectRoot, "environments", "*.json"))
	if err != nil {
		errs = append(errs, err)
	}
	// Glob swallows read errors, so surface an unreadable environments dir.
	if _, err := os.ReadDir(filepath.Join(projectRoot, "environments")); err != nil && !os.IsNotExist(err) {
		errs = append(errs, err)
	}
	for _, p := range append(paths, envs...) {
		if err := tightenFile(p); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func tightenFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 == 0 {
		return nil
	}
	return os.Chmod(path, 0o600)
}
