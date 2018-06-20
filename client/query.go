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
	"fmt"
	"regexp"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/gnmi/path"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// NotificationHandler is a type for the client specific handler function.
//
// Client implementations will pass all kinds of received notifications as they
// arrive.
type NotificationHandler func(Notification) error

// ProtoHandler is a type for the raw handling of the RPC layer. Most users
// should use NotificationHandler instead.
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

	// Pre-compiled regex to match ASCII characters between [\x20-\x7E]
	// i.e., printable ASCII characters and space
	// https://github.com/grpc/blob/master/doc/PROTOCOL-HTTP2.md
	printableASCII = regexp.MustCompile(`^[\x20-\x7E]*$`).MatchString
)

// Destination contains data used to connect to a server.
type Destination struct {
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
	// Timeout is the connection timeout for the query. It will *not* prevent a
	// slow (or streaming) query from completing, this only affects the initial
	// connection and broken connection detection.
	//
	// If Timeout is not set, default is 1 minute.
	Timeout time.Duration
	// Credentials are used for authentication with the target. Optional.
	Credentials *Credentials
	// TLS config to use when connecting to target. Optional.
	TLS *tls.Config
	// Extra contains arbitrary additional metadata to be passed to the
	// target. Optional.
	Extra map[string]string
}

// Validate validates the fields of Destination.
func (d Destination) Validate() error {
	if len(d.Addrs) == 0 {
		return errors.New("Destination.Addrs is empty")
	}
	if d.Credentials != nil {
		return d.Credentials.validate()
	}
	return nil
}

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
	// UpdatesOnly will only stream incremental updates rather than providing the
	// client with an initial snapshot.  This again is implementation specific
	// if the agent doesn't not accept that query it is up the client library to
	// decide wheter to return an error or to make a normal subscription then
	// ignore the initial sync and only provide increment updates.
	UpdatesOnly bool
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
	// SubReq is an optional field. If not nil, gnmi client implementation uses
	// it rather than generating from client.Query while sending gnmi Subscribe RPC.
	SubReq *gpb.SubscribeRequest
}

// Destination extracts a Destination instance out of Query fields.
//
// Ideally we would embed Destination in Query. But in order to not break the
// existing API we have this workaround.
func (q Query) Destination() Destination {
	return Destination{
		Addrs:       q.Addrs,
		Target:      q.Target,
		Replica:     q.Replica,
		Timeout:     q.Timeout,
		Credentials: q.Credentials,
		TLS:         q.TLS,
		Extra:       q.Extra,
	}
}

// Credentials contains information necessary to authenticate with the target.
// Currently only username/password are supported, but may contain TLS client
// certificate in the future.
type Credentials struct {
	Username string
	Password string
}

// Validates the credentials against printable ASCII characters
func (c Credentials) validate() error {
	if !printableASCII(c.Username) {
		return errors.New("Credentials.Username contains non printable ASCII characters")
	}
	if !printableASCII(c.Password) {
		return errors.New("Credentials.Password contains non printable ASCII characters")
	}
	return nil
}

// NewQuery returns a populated Query from given gnmi SubscribeRequest.
// Query fields that are not part of SubscribeRequest must be set on
// the returned object.
// During transtion to support only gnmi, having Query and SubscribeRequest
// in sync is important. There are two approaches to ensure that; one is
// validating whether Query and SubscribeRequest are same after they are set, the other is
// populating the fields of Query from SubscribeRequest and filling out the rest
// on the returned object. NewQuery embraces the latter option.
func NewQuery(sr *gpb.SubscribeRequest) (Query, error) {
	q := Query{}
	if sr == nil {
		return q, errors.New("input is nil")
	}

	s, ok := sr.Request.(*gpb.SubscribeRequest_Subscribe)
	if !ok {
		return q, fmt.Errorf("got %T, want gpb.SubscribeRequest_Subscribe as input", sr)
	}

	if s.Subscribe == nil {
		return q, errors.New("Subscribe field in SubscribeRequest_Subscribe is nil")
	}

	if s.Subscribe.Prefix == nil {
		return q, errors.New("Prefix field in SubscriptionList is nil")
	}
	q.Target = s.Subscribe.Prefix.Target
	q.UpdatesOnly = s.Subscribe.UpdatesOnly
	switch s.Subscribe.Mode {
	case gpb.SubscriptionList_ONCE:
		q.Type = Once
	case gpb.SubscriptionList_POLL:
		q.Type = Poll
	case gpb.SubscriptionList_STREAM:
		q.Type = Stream
	}

	for _, su := range s.Subscribe.Subscription {
		q.Queries = append(q.Queries, path.ToStrings(su.Path, false))
	}
	q.SubReq = sr

	return q, nil
}

// Validate validates that query contains valid values that any client should
// be able use to form a valid backend request.
func (q Query) Validate() error {
	if err := q.Destination().Validate(); err != nil {
		return err
	}
	switch {
	case q.Type == Unknown:
		return errors.New("Query type cannot be Unknown")
	case len(q.Queries) == 0:
		return errors.New("Query.Queries not set")
	case q.NotificationHandler != nil && q.ProtoHandler != nil:
		return errors.New("only one of Notification or ProtoHandler must be set")
	case q.NotificationHandler == nil && q.ProtoHandler == nil:
		return errors.New("one of Notification or ProtoHandler must be set")
	}
	return nil
}
