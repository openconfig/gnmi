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

import (
	"crypto/tls"
	"errors"
	"time"

	"github.com/golang/protobuf/proto"
)

// NotificationHandler is a type for the client specific handler function. It
// is passed from the generic client to the implementation specific library.
// It provides a callback for each notification handled by the client specific
// implementation.
type NotificationHandler func(Notification) error

// ProtoHandler is a type for the raw handling of the RPC layer.  It is exposed
// here only for very lower level interactions and should only be used when
// developing a debugging client.
type ProtoHandler func(proto.Message) error

// Type defines the type of query in a Query.
type Type int

// NewType returns a new QueryType based on the provided string.
func NewType(s string) Type {
	v, ok := typeConst[s]
	if !ok {
		return Unknown
	}
	return v
}

// String returns the string representation of the QueryType.
func (q Type) String() string {
	return typeString[q]
}

const (
	// Unknown is an unknown query and should always be treated as an error.
	Unknown Type = iota
	// Once will perform a Once query against the agent.
	Once
	// Poll will perform a Polling query against the agent.
	Poll
	// Stream will perform a Streaming query against the agent.
	Stream
)

var (
	typeString = map[Type]string{
		Unknown: "unknown",
		Once:    "once",
		Poll:    "poll",
		Stream:  "stream",
	}

	typeConst = map[string]Type{
		"unknown": Unknown,
		"once":    Once,
		"poll":    Poll,
		"stream":  Stream,
	}
)

// Query contains all of the parameters necessary to initiate the query.
type Query struct {
	// Addrs is a slice of addresses by which a target may be reached. Most
	// clients will only handle the first element.
	Addrs []string
	// Target is the target of the query.  Maybe empty if the query is performed
	// against an end target vs. a collector.
	Target string
	// Replica is the specific backend to contact.  This field is implementation
	// specific and for direct agent communication should not be set. default is
	// first available.
	Replica int
	// Discard will only stream incremental updates rather than providing the
	// client with an initial snapshot.  This again is implementation specific
	// if the agent doesn't not accept that query it is up the client library to
	// decide wheter to return an error or to make a normal subscription then
	// ignore the initial sync and only provide increment updates.
	Discard bool
	// Queries contains the list of Paths to query.
	Queries []Path
	// Type of query to perform.
	Type Type
	// Timeout is the connection timeout for the query. It will *not* prevent a
	// slow (or streaming) query from completing, this only affects the initial
	// connection and broken connection detection.
	//
	// If Timeout is not set, default is 1 minute.
	Timeout time.Duration
	// NotificationHandler is the per notification callback handed to a vendor
	// specific implementation. For every notificaiton this call back will be
	// called.
	NotificationHandler NotificationHandler
	// ProtoHandler, if set, will receive all response protos sent by the
	// backend. Only one of NotificationHandler or ProtoHandler may be
	// set.
	ProtoHandler ProtoHandler
	// Credentials are used for authentication with the target. Optional.
	Credentials *Credentials
	// TLS config to use when connecting to target. Optional.
	TLS *tls.Config
	// Extra contains arbitrary additional metadata to be passed to the
	// target. Optional.
	Extra map[string]string
}

// Credentials contains information necessary to authenticate with the target.
// Currently only username/password are supported, but may contain TLS client
// certificate in the future.
type Credentials struct {
	Username string
	Password string
}

// Validate validates that query contains valid values that any client should
// be able use to form a valid backend request.
func (q *Query) Validate() error {
	switch {
	default:
		return nil
	case q == nil:
		return errors.New("Query cannot be nil")
	case q.Type == Unknown:
		return errors.New("Query type cannot be Unknown")
	case len(q.Addrs) == 0:
		return errors.New("Query.Addrs not set")
	case len(q.Queries) == 0:
		return errors.New("Query.Queries not set")
	case q.NotificationHandler != nil && q.ProtoHandler != nil:
		return errors.New("only one of Notification or ProtoHandler must be set")
	case q.NotificationHandler == nil && q.ProtoHandler == nil:
		return errors.New("one of Notification or ProtoHandler must be set")
	}
}

// SetRequest contains slices of changes to apply.
//
// The difference between Update and Replace is that on non-primitive values
// (lists/maps), Update acts as "append"; it only overwrites an existing value
// in the container when the key matches.
// Replace overwrites the entire container regardless of current contents.
type SetRequest struct {
	Delete  []Path
	Update  []Leaf
	Replace []Leaf
}

// SetResponse contains the timestamp of an applied SetRequest.
type SetResponse struct {
	TS time.Time
}
