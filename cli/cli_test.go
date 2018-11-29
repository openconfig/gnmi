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

package cli

import (
	"context"
	"crypto/tls"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"github.com/openconfig/gnmi/client"
	_ "github.com/openconfig/gnmi/client/gnmi"
	"github.com/openconfig/gnmi/testing/fake/gnmi"
	"github.com/openconfig/gnmi/testing/fake/testing/grpc/config"

	fpb "github.com/openconfig/gnmi/testing/fake/proto"
)

func TestSendQueryAndDisplayFail(t *testing.T) {
	tests := []struct {
		desc string
		q    client.Query
		cfg  *Config
		want string
	}{{
		desc: "addr not set",
		want: "Destination.Addrs is empty",
		q:    client.Query{Queries: []client.Path{{"a"}}, Type: client.Once},
		cfg:  &Config{DisplayType: "group"},
	}, {
		desc: "invalid display",
		want: `unknown display type "invalid display"`,
		q:    client.Query{Queries: []client.Path{{"a"}}, Type: client.Once},
		cfg:  &Config{DisplayType: "invalid display"},
	}, {
		desc: "unknown query type",
		want: `unknown query type`,
		q:    client.Query{Queries: []client.Path{{"a"}}, Type: client.Unknown},
		cfg:  &Config{DisplayType: "group"},
	}}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if err := sendQueryAndDisplay(context.Background(), tt.q, tt.cfg); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("sendQueryAndDisplay(ctx, %v, %v): unexpected error, got %v want error contains %q", tt.q, tt.cfg, err, tt.want)
			}
		})
	}
}

func TestSendQueryAndDisplay(t *testing.T) {
	var displayOut string
	display := func(b []byte) {
		displayOut += string(b) + "\n"
	}
	tests := []struct {
		desc    string
		updates []*fpb.Value
		query   client.Query
		cfg     Config
		want    string
		sort    bool
	}{{
		desc: "single target single output with provided layout",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Delimiter:     "/",
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "single",
			Timestamp:     "2006-01-02-15:04:05",
		},
		want: `dev1/a/b, 5, 1969-12-31-16:00:00
`,
	}, {
		desc: "single target single output with timestamp on",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Delimiter:     "/",
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "single",
			Timestamp:     "on",
		},
		want: `dev1/a/b, 5, 1969-12-31-16:00:00.000000100
`,
	}, {
		desc: "single target single output",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Delimiter:     "/",
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "single",
		},
		want: `dev1/a/b, 5
`,
	}, {
		desc: "single target group output",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "group",
		},
		want: `{
  "dev1": {
    "a": {
      "b": 5
    }
  }
}
`,
	}, {
		desc: "single target single output with peer",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "group",
			DisplayPeer:   true,
		},
		want: `// CLI peer: 127.0.0.1:port
{
  "dev1": {
    "a": {
      "b": 5
    }
  }
}
`,
	}, {
		desc: "single target single output with latency",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Delimiter:     "/",
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "single",
			Latency:       true,
		},
		want: `dev1/a/b, 5, <h>h<m>m<s>.<ns>s
`,
	}, {
		desc: "single target single output with delete",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Path: []string{"a", "c"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Path: []string{"a", "c"}, Value: &fpb.Value_Delete{Delete: &fpb.DeleteValue{}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 200}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Delimiter:     "/",
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "single",
		},
		want: `dev1/a/b, 5
dev1/a/c, 5
dev1/a/c, <nil>
`,
	}, {
		desc: "single target multiple paths",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			// The following is disallowed because the tree already has a branch where
			// this leaf requests to be, thus dropped.
			{Path: []string{"a"}, Value: &fpb.Value_StringValue{StringValue: &fpb.StringValue{Value: "foo"}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 105}},
			// The following is disallowed because the tree already has a leaf where
			// this branch requests to be, thus dropped.
			{Path: []string{"a", "b", "c"}, Value: &fpb.Value_DoubleValue{DoubleValue: &fpb.DoubleValue{Value: 3.14}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "group",
		},
		want: `{
  "dev1": {
    "a": {
      "b": 5
    }
  }
}
`,
	}, {
		desc: "single target single paths (proto)",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 1440807212000000000}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Display:     display,
			DisplayType: "proto",
		},
		want: `sync_response: true

update: <
  timestamp: 1440807212000000000
  prefix: <
    target: "dev1"
  >
  update: <
    path: <
      element: "a"
      element: "b"
    >
    val: <
      int_val: 5
    >
  >
>

`,
	}, {
		desc: "single target single paths (proto) with size",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 1440807212000000000}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Display:     display,
			DisplayType: "proto",
			DisplaySize: true,
		},
		want: `sync_response: true

update: <
  timestamp: 1440807212000000000
  prefix: <
    target: "dev1"
  >
  update: <
    path: <
      element: "a"
      element: "b"
    >
    val: <
      int_val: 5
    >
  >
>

// total response size: 36
`,
	}, {
		desc: "single target multiple paths (with timestamp)",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 1440807212000000000}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 1440807212000000001}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "group",
			Timestamp:     "on",
		},
		want: `{
  "dev1": {
    "a": {
      "b": {
        "timestamp": "2015-08-28-17:13:32.000000000",
        "value": 5
      }
    }
  }
}
`,
	}, {
		desc: "single target multiple paths (with raw timestamp)",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "group",
			Timestamp:     "raw",
		},
		want: `{
  "dev1": {
    "a": {
      "b": {
        "timestamp": 100,
        "value": 5
      }
    }
  }
}
`,
	}, {
		desc: "single target multiple paths (with DoW timestamp)",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 1440807212000000000}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 1440807212000000001}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "group",
			Timestamp:     "Monday",
		},
		want: `{
  "dev1": {
    "a": {
      "b": {
        "timestamp": "Friday",
        "value": 5
      }
    }
  }
}
`,
	}, {
		desc: "single target multiple root paths",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Path: []string{"a", "c"}, Value: &fpb.Value_DoubleValue{DoubleValue: &fpb.DoubleValue{Value: 3.25}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 101}},
			{Path: []string{"d", "e", "f"}, Value: &fpb.Value_StringValue{StringValue: &fpb.StringValue{Value: "foo"}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 102}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"*"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "group",
		},
		want: `{
  "dev1": {
    "a": {
      "b": 5,
      "c": 3.25
    },
    "d": {
      "e": {
        "f": "foo"
      }
    }
  }
}
`,
	}, {
		desc: "single target multiple paths (Pollingx2)",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Poll,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Display:         display,
			DisplayPrefix:   "",
			DisplayIndent:   "  ",
			DisplayType:     "group",
			Count:           2,
			PollingInterval: 100 * time.Millisecond,
		},
		want: `{
  "dev1": {
    "a": {
      "b": 5
    }
  }
}
{
  "dev1": {
    "a": {
      "b": 5
    }
  }
}
`,
	}, {
		desc: "single target multiple paths (streaming)",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Stream,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "group",
			// StreamingDuration will expire before Count updates are received because
			// no updates are being streamed in the test.
			Count:             2,
			StreamingDuration: 100 * time.Millisecond,
		},
		want: `{
  "dev1": {
    "a": {
      "b": 5
    }
  }
}
`,
	}, {
		desc: "single target multiple paths (single line)",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Path: []string{"a", "c"}, Value: &fpb.Value_DoubleValue{DoubleValue: &fpb.DoubleValue{Value: 3.25}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 101}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Delimiter:     "/",
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "single",
		},
		want: `dev1/a/b, 5
dev1/a/c, 3.25
`,
		sort: true,
	}, {
		desc: "single target multiple paths (single line raw)",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Path: []string{"a", "c"}, Value: &fpb.Value_DoubleValue{DoubleValue: &fpb.DoubleValue{Value: 3.25}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 101}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Delimiter:     ".",
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "single",
			Timestamp:     "raw",
		},
		want: `dev1.a.b, 5, 100
dev1.a.c, 3.25, 101
`,
		sort: true,
	}, {
		desc: "single target multiple paths (proto short)",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Display:     display,
			DisplayType: "shortproto",
		},
		want: `update:<timestamp:100 prefix:<target:"dev1" > update:<path:<element:"a" element:"b" > val:<int_val:5 > > >
sync_response:true
`,
	}, {
		desc: "single target multiple paths (with display size)",
		updates: []*fpb.Value{
			{Path: []string{"a", "b"}, Value: &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 100}},
			{Value: &fpb.Value_Sync{Sync: 1}, Repeat: 1, Timestamp: &fpb.Timestamp{Timestamp: 300}},
		},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Display:       display,
			DisplayIndent: "  ",
			DisplayType:   "group",
			DisplaySize:   true,
		},
		want: `{
  "dev1": {
    "a": {
      "b": 5
    }
  }
}
// total response size: 49
`,
	}}

	opt, err := config.WithSelfTLSCert()
	if err != nil {
		t.Fatalf("failed to generate cert: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			displayOut = ""
			s, err := gnmi.New(
				&fpb.Config{
					Target:      "dev1",
					DisableSync: true,
					Values:      tt.updates,
				},
				[]grpc.ServerOption{opt},
			)
			if err != nil {
				t.Fatal("failed to start test server")
			}
			defer s.Close()

			tt.query.Addrs = []string{s.Address()}

			if err := QueryDisplay(context.Background(), tt.query, &tt.cfg); err != nil {
				// This is fine if we use cfg.StreamingDuration.
				t.Logf("sendQueryAndDisplay returned error: %v", err)
			}

			// The test server comes up on an arbitrary port.  Remove the port number
			// from the output before comparison.
			re := regexp.MustCompile("localhost:[0-9]*")
			got := re.ReplaceAllLiteralString(displayOut, "127.0.0.1:port")
			reLat := regexp.MustCompile(`\d+h\d+m\d+.\d+s`)
			got = reLat.ReplaceAllLiteralString(got, "<h>h<m>m<s>.<ns>s")
			if tt.sort {
				lines := strings.Split(got, "\n")
				sort.Strings(lines)
				// move blank line to end
				lines = append(lines[1:], lines[0])
				got = strings.Join(lines, "\n")
			}

			if got != tt.want {
				t.Errorf("sendQueryAndDisplay(ctx, address, %v, %v):\ngot(%d):\n%s\nwant(%d):\n%s", tt.query, tt.cfg, len(got), got, len(tt.want), tt.want)
			}
		})
	}
}

func TestGNMIClient(t *testing.T) {
	var displayOut string
	display := func(b []byte) {
		displayOut += string(b) + "\n"
	}
	tests := []struct {
		desc    string
		updates []*fpb.Value
		query   client.Query
		cfg     Config
		want    string
		sort    bool
	}{{
		desc: "single target single output with provided layout",
		updates: []*fpb.Value{{
			Path:      []string{"a"},
			Timestamp: &fpb.Timestamp{Timestamp: 100},
			Repeat:    1,
			Value:     &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}},
		}, {
			Path:      []string{"a", "b"},
			Timestamp: &fpb.Timestamp{Timestamp: 100},
			Repeat:    1,
			Value:     &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}},
		}, {
			Path:      []string{"a", "b"},
			Timestamp: &fpb.Timestamp{Timestamp: 200},
			Repeat:    1,
			Value:     &fpb.Value_Delete{Delete: &fpb.DeleteValue{}},
		}, {
			Timestamp: &fpb.Timestamp{Timestamp: 300},
			Repeat:    1,
			Value:     &fpb.Value_Sync{Sync: 1},
		}},
		query: client.Query{
			Target:  "dev",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			Timeout: 3 * time.Second,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Delimiter:     "/",
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "single",
			Timestamp:     "2006-01-02-15:04:05",
		},
		want: `dev/a, 5, 1969-12-31-16:00:00
dev/a/b, 5, 1969-12-31-16:00:00
dev/a/b, <nil>, 1969-12-31-16:00:00
`,
	}, {
		desc: "single target group output with provided layout",
		updates: []*fpb.Value{{
			Path:      []string{"a"},
			Timestamp: &fpb.Timestamp{Timestamp: 100},
			Repeat:    1,
			Value:     &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}},
		}, {
			Path:      []string{"a", "b"},
			Timestamp: &fpb.Timestamp{Timestamp: 100},
			Repeat:    1,
			Value:     &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}},
		}, {
			Path:      []string{"a", "b"},
			Timestamp: &fpb.Timestamp{Timestamp: 200},
			Repeat:    1,
			Value:     &fpb.Value_Delete{Delete: &fpb.DeleteValue{}},
		}, {
			Timestamp: &fpb.Timestamp{Timestamp: 300},
			Repeat:    1,
			Value:     &fpb.Value_Sync{Sync: 1},
		}},
		query: client.Query{
			Target:  "dev",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			Timeout: 3 * time.Second,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Delimiter:     "/",
			Display:       display,
			DisplayPrefix: "",
			DisplayIndent: "  ",
			DisplayType:   "group",
			Timestamp:     "2006-01-02-15:04:05",
		},
		want: `{
  "dev": {
    "a": {
      "timestamp": "1969-12-31-16:00:00",
      "value": 5
    }
  }
}
`,
	}, {
		desc: "single target multiple paths (proto short)",
		updates: []*fpb.Value{{
			Path:      []string{"a"},
			Timestamp: &fpb.Timestamp{Timestamp: 100},
			Repeat:    1,
			Value:     &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}},
		}, {
			Path:      []string{"a", "b"},
			Timestamp: &fpb.Timestamp{Timestamp: 100},
			Repeat:    1,
			Value:     &fpb.Value_IntValue{IntValue: &fpb.IntValue{Value: 5}},
		}, {
			Path:      []string{"a", "b"},
			Timestamp: &fpb.Timestamp{Timestamp: 200},
			Repeat:    1,
			Value:     &fpb.Value_Delete{Delete: &fpb.DeleteValue{}},
		}, {
			Timestamp: &fpb.Timestamp{Timestamp: 300},
			Repeat:    1,
			Value:     &fpb.Value_Sync{Sync: 1},
		}},
		query: client.Query{
			Target:  "dev1",
			Queries: []client.Path{{"a"}},
			Type:    client.Once,
			Timeout: 3 * time.Second,
			TLS:     &tls.Config{InsecureSkipVerify: true},
		},
		cfg: Config{
			Display:     display,
			DisplayType: "shortproto",
		},
		want: `update:<timestamp:100 prefix:<target:"dev1" > update:<path:<element:"a" > val:<int_val:5 > > >
update:<timestamp:100 prefix:<target:"dev1" > update:<path:<element:"a" element:"b" > val:<int_val:5 > > >
update:<timestamp:200 prefix:<target:"dev1" > delete:<element:"a" element:"b" > >
sync_response:true
`,
	}}
	opt, err := config.WithSelfTLSCert()
	if err != nil {
		t.Fatalf("failed to generate cert: %v", err)
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			displayOut = ""
			s, err := gnmi.New(
				&fpb.Config{
					Target:      "dev1",
					DisableSync: true,
					Values:      tt.updates,
				},
				[]grpc.ServerOption{opt},
			)
			if err != nil {
				t.Fatal("failed to start test server")
			}
			defer s.Close()

			tt.query.Addrs = []string{s.Address()}
			if err := QueryDisplay(context.Background(), tt.query, &tt.cfg); err != nil {
				// This is fine if we use cfg.StreamingDuration.
				t.Logf("sendQueryAndDisplay returned error: %v", err)
			}

			// The test server comes up on an arbitrary port.  Remove the port number
			// from the output before comparison.
			re := regexp.MustCompile("127.0.0.1:[0-9]*")
			got := re.ReplaceAllLiteralString(displayOut, "127.0.0.1:port")
			reLat := regexp.MustCompile(`\d+h\d+m\d+.\d+s`)
			got = reLat.ReplaceAllLiteralString(got, "<h>h<m>m<s>.<ns>s")
			if got != tt.want {
				t.Errorf("sendQueryAndDisplay(ctx, address, %v, %v):\ngot(%d):\n%s\nwant(%d):\n%s", tt.query, tt.cfg, len(got), got, len(tt.want), tt.want)
			}
		})
	}
}
