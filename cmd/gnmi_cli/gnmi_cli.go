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

// The gnmi_cli program implements the GNMI CLI.
//
// usage:
// gnmi_cli --address=<ADDRESS>                            \
//            -q=<OPENCONFIG_PATH[,OPENCONFIG_PATH[,...]]> \
//            [-qt=<QUERY_TYPE>]                           \
//            [-<ADDITIONAL_OPTION(s)>]
package main

import (
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"flag"
	log "github.com/golang/glog"
	"context"
	"golang.org/x/crypto/ssh/terminal"
	"github.com/openconfig/gnmi/cli"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/client/flags"

	// Register supported client types.
	_ "github.com/openconfig/gnmi/client/gnmi"
	_ "github.com/openconfig/gnmi/client/openconfig"
)

var (
	q   client.Query
	mu  sync.Mutex
	cfg = cli.Config{Display: func(b []byte) {
		defer mu.Unlock()
		mu.Lock()
		os.Stdout.Write(append(b, '\n'))
	}}

	clientTypes = flags.NewStringList(&cfg.ClientTypes, nil)
	queryFlag   = &flags.StringList{}
	queryType   = flag.String("query_type", client.Once.String(), "Type of result, one of: (o, once, p, polling, s, streaming).")
	queryAddr   = flags.NewStringList(&q.Addrs, nil)

	setFlag  = flag.Bool("set", false, "When set, CLI will perform a Set request. At least one of --delete/--update/--replace must be set.")
	deletes  = &flags.StringList{}
	updates  = &flags.StringMap{}
	replaces = &flags.StringMap{}

	withUserPass = flag.Bool("with_user_pass", false, "When set, CLI will prompt for username/password to use when connecting to a target.")
)

func init() {
	flag.Var(clientTypes, "client_types", fmt.Sprintf("List of explicit client types to attempt: (%s) (default: attempt all registered clients).", strings.Join(client.RegisteredImpls(), ", ")))
	flag.Var(queryFlag, "query", "Comma separated list of queries.  Each query is a delimited list of OpenConfig path nodes which may also be specified as a glob (*).  The delimeter can be specified with the --delimiter flag.")
	// Query command-line flags.
	flag.Var(queryAddr, "address", "Address of the GNMI target to query.")
	flag.BoolVar(&q.Discard, "discard", false, "Discard the initial sync and only stream updates. Setting this flag for once or polling queries will cause nothing to be returned.")
	// Config command-line flags.
	flag.DurationVar(&cfg.PollingInterval, "polling_interval", 30*time.Second, "Interval at which to poll in seconds if polling is specified for query_type.")
	flag.UintVar(&cfg.Count, "count", 0, "Number of polling/streaming events (0 is infinite).")
	flag.StringVar(&cfg.Delimiter, "delimiter", "/", "Delimiter between path nodes in query.")
	flag.DurationVar(&cfg.StreamingDuration, "streaming_duration", 0, "Length of time to collect streaming queries (0 is infinite).")
	flag.StringVar(&cfg.DisplayPrefix, "display_prefix", "", "Per output line prefix.")
	flag.StringVar(&cfg.DisplayIndent, "display_indent", "  ", "Output line, per nesting-level indent.")
	flag.StringVar(&cfg.DisplayType, "display_type", "group", "Display output type (g, group, s, single, p, proto).")
	flag.DurationVar(&q.Timeout, "timeout", 30*time.Second, "Terminate query if no RPC is established within the timeout duration.")
	flag.StringVar(&cfg.Timestamp, "timestamp", "", "Specify timestamp formatting in output.  One of (<empty string>, on, raw, <FORMAT>) where <empty string> is disabled, on is human readable, raw is int64 nanos since epoch, and <FORMAT> is according to golang time.Format(<FORMAT>)")
	flag.BoolVar(&cfg.DisplaySize, "display_size", false, "Display the total size of query response.")
	flag.BoolVar(&cfg.Latency, "latency", false, "Display the latency for receiving each update (Now - update timestamp).")
	flag.Var(deletes, "delete", "List of paths to delete; --set flag must be set.")
	flag.Var(updates, "update", "List of paths to update; --set flag must be set.")
	flag.Var(replaces, "replace", "List of paths to replace; --set flag must be set.")

	// Shortcut flags that can be used in place of the longform flags above.
	flag.Var(queryAddr, "a", "Short for address.")
	flag.Var(queryFlag, "q", "Short for query.")
	flag.UintVar(&cfg.Count, "c", cfg.Count, "Short for count.")
	flag.StringVar(&cfg.Delimiter, "d", cfg.Delimiter, "Short for delimiter.")
	flag.StringVar(&cfg.Timestamp, "ts", cfg.Timestamp, "Short for timestamp.")
	flag.StringVar(queryType, "qt", *queryType, "Short for query_type.")
	flag.StringVar(&cfg.DisplayType, "dt", cfg.DisplayType, "Short for display_type.")
	flag.DurationVar(&cfg.StreamingDuration, "sd", cfg.StreamingDuration, "Short for streaming_duration.")
	flag.DurationVar(&cfg.PollingInterval, "pi", cfg.PollingInterval, "Short for polling_interval.")
	flag.BoolVar(&cfg.DisplaySize, "ds", cfg.DisplaySize, "Short for display_size.")
	flag.BoolVar(&cfg.Latency, "l", cfg.Latency, "Short for latency.")
}

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	// Terminate immediately on Ctrl+C, skipping lame-duck mode.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		cancel()
	}()

	if len(q.Addrs) == 0 {
		log.Exit("--address must be set")
	}
	if *withUserPass {
		var err error
		q.Credentials, err = readCredentials()
		if err != nil {
			log.Exit(err)
		}
	}
	// Transport security is required even with PerRPCCredentials. Make this
	// insecure for now.
	q.TLS = &tls.Config{InsecureSkipVerify: true}

	if *setFlag {
		if err := executeSet(ctx); err != nil {
			log.Error(err)
		}
		return
	}
	if q.Type = cli.QueryType(*queryType); q.Type == client.Unknown {
		log.Exit("--query_type must be one of: (o, once, p, polling, s, streaming)")
	}
	// Parse queryFlag into appropriate format.
	if len(*queryFlag) == 0 {
		log.Exit("--query must be set")
	}
	for _, path := range *queryFlag {
		q.Queries = append(q.Queries, strings.Split(path, cfg.Delimiter))
	}

	if err := cli.QueryDisplay(ctx, q, &cfg); err != nil {
		log.Errorf("cli.QueryDisplay:\n\t%v", err)
	}
}

func executeSet(ctx context.Context) error {
	req := client.SetRequest{}

	for _, p := range *deletes {
		req.Delete = append(req.Delete, strings.Split(p, cfg.Delimiter))
	}

	for p, v := range *updates {
		req.Update = append(req.Update, client.Leaf{
			Path: strings.Split(p, cfg.Delimiter),
			Val:  v,
		})
	}
	for p, v := range *replaces {
		req.Replace = append(req.Replace, client.Leaf{
			Path: strings.Split(p, cfg.Delimiter),
			Val:  v,
		})
	}

	return cli.Set(ctx, req, &cfg)
}

func readCredentials() (*client.Credentials, error) {
	c := &client.Credentials{}

	fmt.Print("username: ")
	_, err := fmt.Scan(&c.Username)
	if err != nil {
		return nil, err
	}

	fmt.Print("password: ")
	pass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	c.Password = string(pass)

	return c, nil
}
