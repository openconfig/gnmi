/*
Copyright 2017 Google Inc.

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

package client

// Notification are internal messages used for abstracting protocol specific
// messages for clients. isNotification is only present to force typing
// assertions.
type Notification interface {
	isNotification()
}

// Update is an update to the leaf in the tree.
type Update Leaf

func (u Update) isNotification() {}

// Delete is an explicit delete of the path in the tree.
type Delete Leaf

func (d Delete) isNotification() {}

// Error is a inband error notification. It could be received without breaking
// the query or connection.
type Error struct {
	s string
}

// NewError will return a new error with the provided text.
func NewError(s string) Error {
	return Error{s: s}
}

// Error is provided to implement the error interface.
func (e Error) Error() string {
	return e.s
}
func (e Error) isNotification() {}

// Sync is an inband notification that the client has sent everything in it's
// cache at least once. This does not mean EVERYTHING you wanted is there only
// that the target has sent everything it currently has... which may be nothing.
type Sync struct{}

func (s Sync) isNotification() {}

// Connected is a synthetic notification sent when connection is established.
// It's sent before any other notifications on a new client.
type Connected struct{}

func (s Connected) isNotification() {}
