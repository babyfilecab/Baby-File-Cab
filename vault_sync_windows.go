//go:build windows

package main

import "os"

// Windows does not support os.File.Sync on a directory handle opened by os.Open.
// File contents are still synced by writeAtomicPrivate before rename. SQLite
// retains its own transaction durability. Validate the directory here rather
// than failing every vault write with Access Denied. Directory-entry durability
// across sudden power loss is weaker than the Unix fsync path.
func syncVaultDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return &os.PathError{Op: "sync directory", Path: path, Err: os.ErrInvalid}
	}
	return nil
}
