// Package watch detects raw changes to files.
package watch

import (
	"context"
)

// Watcher watches files at the given paths for changes.
type Watcher interface {
	// Read blocks and returns the next update for a file. An error is returned
	// when the file cannot be read. Subsequent calls block until the underlying
	// contents or error changes. When multiple updates have occurred for a file,
	// Read coalesces and returns the latest update.
	//
	// If ctx is cancelled, Read returns an error.
	Read(ctx context.Context) (Update, error)

	// Add causes Watcher to monitor an additional file. The format is
	// filesystem-specific. If Close has been called, this has no effect.
	Add(path string) error

	// Remove causes Watcher to stop monitoring a file. The path must match one
	// already monitored in the same format. The format is filesystem-specific.
	Remove(path string) error

	// Close causes Watcher to stop watching all files and release its resources.
	Close()
}

// Update represents contents for a file. Path can represent fan-out on
// individual files when watching a directory path.
type Update struct {
	Path     string
	Contents []byte
}
