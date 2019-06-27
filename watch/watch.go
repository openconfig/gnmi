/*
Copyright 2019 Google Inc.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package watch detects raw changes to files. The interpretation of a file is
// filesystem-specific.
package watch

import (
	"context"
)

// Watcher watches files at the given paths for changes.
type Watcher interface {
	// Read blocks and returns the next update for a file.
	//
	// An error is returned when there is an underyling issue in the Watcher
	// preventing Read, or ctx is cancelled. The returned error may indicate a
	// fatal issue requiring a new Watcher to be created.
	//
	// Subsequent calls block until the underlying
	// contents or error changes. When multiple updates have occurred for a file,
	// Read coalesces and returns the latest update.
	Read(ctx context.Context) (Update, error)

	// Add causes Watcher to monitor an additional path. The format is
	// filesystem-specific. If Close has been called, this has no effect.
	Add(path string) error

	// Remove causes Watcher to stop monitoring a path. The path must match one
	// already monitored in the same format. The format is filesystem-specific.
	Remove(path string) error

	// Close causes Watcher to stop watching all files and release its resources.
	Close()
}

// Update represents contents for a file. Path can represent fan-out on
// individual files when watching a path that contains multiple files.
// Update contains an error reflecting path-specific problems.
type Update struct {
	Path     string
	Contents []byte
	Err      error
}
