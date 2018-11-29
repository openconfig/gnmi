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

// Package cli provides the query capabilities for streaming telemetry.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/ctree"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

const layout = "2006-01-02-15:04:05.000000000"

var (
	queryTypeMap = map[string]client.Type{
		"o": client.Once, "once": client.Once, "ONCE": client.Once,
		"p": client.Poll, "polling": client.Poll, "POLLING": client.Poll,
		"s": client.Stream, "streaming": client.Stream, "STREAMING": client.Stream,
	}
	displayTypeMap = map[string]string{
		"g": "GROUP", "group": "GROUP", "GROUP": "GROUP",
		"s": "SINGLE", "single": "SINGLE", "SINGLE": "SINGLE",
		"p": "PROTO", "proto": "PROTO", "PROTO": "PROTO",
		"sp": "SHORTPROTO", "shortproto": "SHORTPROTO", "SHORTPROTO": "SHORTPROTO",
	}
)

// Config is a type to hold parameters that affect how the cli sends and
// displays results.
type Config struct {
	PollingInterval   time.Duration // Duration between polling events.
	StreamingDuration time.Duration // Duration to collect updates, 0 is forever.
	Count             uint          // Number of polling/streaming events, 0 is infinite.
	countExhausted    bool          // Trigger to indicate termination.
	Delimiter         string        // Delimiter between path elements when converted to string.
	Display           func([]byte)  // Function called to display each result.
	DisplayPrefix     string        // Prefix for each line of result output.
	DisplayIndent     string        // Indent per nesting level of result output.
	DisplayType       string        // Display results in selected format, grouped, single, proto.
	DisplayPeer       bool          // Display the immediate connected peer.
	// <empty string> - disable timestamp
	// on - human readable timestamp according to layout
	// raw - int64 nanos since epoch
	// <FORMAT> - human readable timestamp according to <FORMAT>
	Timestamp   string // Formatting of timestamp in result output.
	DisplaySize bool
	Latency     bool     // Show latency to client
	ClientTypes []string // List of client types to try.
}

// QueryType returns a client query type for t after trying aliases for the
// type.
func QueryType(t string) client.Type {
	return queryTypeMap[t]
}

// QueryDisplay constructs a query from the supplied arguments (target, queries,
// queryType), sends as an RPC to the specified destination address and displays
// results with the supplied display function.
func QueryDisplay(ctx context.Context, query client.Query, cfg *Config) error {
	if err := sendQueryAndDisplay(ctx, query, cfg); err != nil {
		return fmt.Errorf("sendQueryAndDisplay(ctx, %+v, %+v):\n\t%v", query, cfg, err)
	}
	return nil
}

// ParseSubscribeProto parses given gNMI SubscribeRequest text proto
// into client.Query.
func ParseSubscribeProto(p string) (client.Query, error) {
	var tq client.Query
	sr := &gpb.SubscribeRequest{}
	if err := proto.UnmarshalText(p, sr); err != nil {
		return tq, err
	}
	return client.NewQuery(sr)
}

// sendQueryAndDisplay directs a query to the specified target. The returned
// results are formatted as a JSON string and passed to Config.Display().
func sendQueryAndDisplay(ctx context.Context, query client.Query, cfg *Config) error {
	cancel := func() {}
	if cfg.StreamingDuration > 0 {
		ctx, cancel = context.WithTimeout(ctx, cfg.StreamingDuration)
		defer cancel()
	}
	switch displayTypeMap[cfg.DisplayType] {
	default:
		return fmt.Errorf("unknown display type %q", cfg.DisplayType)
	case "GROUP":
	case "SINGLE":
		return displaySingleResults(ctx, query, cfg)
	case "PROTO":
		return displayProtoResults(ctx, query, cfg, func(r proto.Message) []byte {
			return []byte(proto.MarshalTextString(r))
		})
	case "SHORTPROTO":
		return displayProtoResults(ctx, query, cfg, func(r proto.Message) []byte {
			// r.String() will add extra whitespace at the end for some reason.
			// Trim it down.
			return bytes.TrimSpace([]byte(r.String()))
		})
	}
	switch query.Type {
	default:
		return fmt.Errorf("unknown query type %v", query.Type)
	case client.Once:
		return displayOnceResults(ctx, query, cfg)
	case client.Poll:
		return displayPollingResults(ctx, query, cfg)
	case client.Stream:
		return displayStreamingResults(ctx, query, cfg)
	}
}

// genHandler takes a provided query and cfg and will build a display function
// that is custom for the client and display options configured. It really
// looks more scary than it is. For each client type and display method you
// need to build a custom handler for display the raw values rather than
// the normal "Leaf" value that client normally works with.
func genHandler(cfg *Config) client.NotificationHandler {
	var buf bytes.Buffer // Reuse the same buffer in either case.
	iDisplay := func(p client.Path, v interface{}, ts time.Time) {
		buf.Reset()
		buf.WriteString(strings.Join(p, cfg.Delimiter))
		buf.WriteString(fmt.Sprintf(", %v", v))
		var t interface{}
		switch cfg.Timestamp {
		default: // Assume user has passed a valid layout for time.Format
			t = ts.Format(cfg.Timestamp)
		case "": // Timestamp disabled.
		case "on":
			t = ts.Format(layout)
		case "raw":
			t = ts.UnixNano()
		}
		if t != nil {
			buf.WriteString(fmt.Sprintf(", %v", t))
		}
		if cfg.Latency {
			buf.WriteString(fmt.Sprintf(", %s", time.Since(ts)))
		}
		cfg.Display(buf.Bytes())
	}
	return func(n client.Notification) error {
		switch v := n.(type) {
		default:
			return fmt.Errorf("invalid type: %#v", v)
		case client.Update:
			iDisplay(v.Path, v.Val, v.TS)
		case client.Delete:
			iDisplay(v.Path, v.Val, v.TS)
		case client.Sync, client.Connected:
		case client.Error:
			return v
		}
		return nil
	}
}

// displaySingleResults displays each key/value pair returned on a single line.
func displaySingleResults(ctx context.Context, query client.Query, cfg *Config) error {
	query.NotificationHandler = genHandler(cfg)
	c := &client.BaseClient{}
	if err := c.Subscribe(ctx, query); err != nil {
		return fmt.Errorf("client had error while displaying results:\n\t%v", err)
	}
	return nil
}

// displayProtoResults displays the raw protos returned for the supplied query.
func displayProtoResults(ctx context.Context, query client.Query, cfg *Config, formatter func(proto.Message) []byte) error {
	var sum int64
	query.ProtoHandler = func(msg proto.Message) error {
		if cfg.DisplaySize {
			sum += int64(proto.Size(msg))
		}
		cfg.Display(formatter(msg))
		return nil
	}
	c := &client.BaseClient{}
	if err := c.Subscribe(ctx, query, cfg.ClientTypes...); err != nil {
		return fmt.Errorf("client had error while displaying results:\n\t%v", err)
	}
	if cfg.DisplaySize {
		cfg.Display([]byte(fmt.Sprintf("// total response size: %d", sum)))
	}
	return nil
}

func displayPeer(c client.Client, cfg *Config) {
	if !cfg.DisplayPeer {
		return
	}
	var peer string
	impl, err := c.Impl()
	if err != nil {
		return
	}
	if v, ok := impl.(interface {
		Peer() string
	}); ok {
		peer = v.Peer()
	}
	if peer == "" {
		cfg.Display([]byte("// No peer found for client"))
		return
	}
	cfg.Display([]byte(fmt.Sprintf("// CLI peer: %s", peer)))
}

// displayOnceResults builds all the results returned for for one application of
// query to the OpenConfig data tree and displays the resulting tree in JSON.
func displayOnceResults(ctx context.Context, query client.Query, cfg *Config) error {
	c := client.New()
	if err := c.Subscribe(ctx, query, cfg.ClientTypes...); err != nil {
		return fmt.Errorf("client had error while displaying results:\n\t%v", err)
	}
	displayPeer(c, cfg)
	displayWalk(query.Target, c, cfg)
	return nil
}

func countComplete(cfg *Config) bool {
	switch {
	case cfg.countExhausted:
		return true
	case cfg.Count == 0:
	default:
		cfg.Count--
		if cfg.Count == 0 {
			cfg.countExhausted = true
		}
	}
	return false
}

// displayPollingResults repeatedly calls displayOnceResults at the requested
// interval.
func displayPollingResults(ctx context.Context, query client.Query, cfg *Config) error {
	c := client.New()
	if err := c.Subscribe(ctx, query, cfg.ClientTypes...); err != nil {
		return fmt.Errorf("client had error while displaying results:\n\t%v", err)
	}
	defer c.Close()
	header := false
	for !countComplete(cfg) {
		if err := c.Poll(); err != nil {
			return fmt.Errorf("client.Poll(): %v", err)
		}
		if !header {
			displayPeer(c, cfg)
			header = true
		}
		displayWalk(query.Target, c, cfg)
		if !cfg.countExhausted {
			time.Sleep(cfg.PollingInterval)
		}
	}
	return nil
}

// displayStreamingResults calls displayOnceResults one time, followed by
// subsequent individual updates as they arrive.
func displayStreamingResults(ctx context.Context, query client.Query, cfg *Config) error {
	c := client.New()
	complete := false
	display := func(path []string, ts time.Time, val interface{}) {
		if !complete {
			return
		}
		b := make(pathmap)
		if cfg.Timestamp != "" {
			b.add(append(path, "timestamp"), ts)
			b.add(append(path, "value"), val)
		} else {
			b.add(path, val)
		}
		result, err := json.MarshalIndent(b, cfg.DisplayPrefix, cfg.DisplayIndent)
		if err != nil {
			cfg.Display([]byte(fmt.Sprintf("Error: failed to marshal result: %v", err)))
			return
		}
		cfg.Display(result)
	}
	query.NotificationHandler = func(n client.Notification) error {
		switch v := n.(type) {
		case client.Update:
			display(v.Path, v.TS, v.Val)
		case client.Delete:
			display(v.Path, v.TS, v.Val)
		case client.Sync:
			displayWalk(query.Target, c, cfg)
			complete = true
		case client.Error:
			cfg.Display([]byte(fmt.Sprintf("Error: %v", v)))
		}
		return nil
	}
	return c.Subscribe(ctx, query, cfg.ClientTypes...)
}

func displayWalk(target string, c *client.CacheClient, cfg *Config) {
	b := make(pathmap)
	var addFunc func(path []string, v client.TreeVal)
	switch cfg.Timestamp {
	default:
		addFunc = func(path []string, v client.TreeVal) {
			b.add(path, map[string]interface{}{
				"value":     v.Val,
				"timestamp": v.TS.Format(cfg.Timestamp),
			})
		}
	case "on":
		addFunc = func(path []string, v client.TreeVal) {
			b.add(path, map[string]interface{}{
				"value":     v.Val,
				"timestamp": v.TS.Format(layout),
			})
		}
	case "raw":
		addFunc = func(path []string, v client.TreeVal) {
			b.add(path, map[string]interface{}{
				"value":     v.Val,
				"timestamp": v.TS.UnixNano(),
			})
		}
	case "off", "":
		addFunc = func(path []string, v client.TreeVal) {
			b.add(path, v.Val)
		}
	}
	c.WalkSorted(func(path []string, _ *ctree.Leaf, value interface{}) {
		switch v := value.(type) {
		default:
			b.add(path, fmt.Sprintf("INVALID NODE %#v", value))
		case *ctree.Tree:
		case client.TreeVal:
			addFunc(path, v)
		}
	})
	result, err := json.MarshalIndent(b, cfg.DisplayPrefix, cfg.DisplayIndent)
	if err != nil {
		cfg.Display([]byte(fmt.Sprintf("Error: failed to marshal result: %v", err)))
		return
	}
	cfg.Display(result)
	if cfg.DisplaySize {
		cfg.Display([]byte(fmt.Sprintf("// total response size: %d", len(result))))
	}
}

type pathmap map[string]interface{}

func (m pathmap) add(path []string, v interface{}) {
	if len(path) == 1 {
		m[path[0]] = v
		return
	}

	mm, ok := m[path[0]]
	if !ok {
		mm = make(pathmap)
	}
	mm.(pathmap).add(path[1:], v)
	m[path[0]] = mm
}
