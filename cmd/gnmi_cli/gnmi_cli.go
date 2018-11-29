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
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"flag"
	
	log "github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh/terminal"
	"github.com/openconfig/gnmi/cli"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/client/flags"
	gclient "github.com/openconfig/gnmi/client/gnmi"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

var (
	q   = client.Query{TLS: &tls.Config{}}
	mu  sync.Mutex
	cfg = cli.Config{Display: func(b []byte) {
		defer mu.Unlock()
		mu.Lock()
		os.Stdout.Write(append(b, '\n'))
	}}

	clientTypes = flags.NewStringList(&cfg.ClientTypes, []string{gclient.Type})
	queryFlag   = &flags.StringList{}
	queryType   = flag.String("query_type", client.Once.String(), "Type of result, one of: (o, once, p, polling, s, streaming).")
	queryAddr   = flags.NewStringList(&q.Addrs, nil)

	reqProto = flag.String("proto", "", "Text proto for gNMI request.")

	capabilitiesFlag = flag.Bool("capabilities", false, `When set, CLI will perform a Capabilities request. Usage: gnmi_cli -capabilities [-proto <gnmi.CapabilityRequest>] -address <address> [other flags ...]`)
	getFlag          = flag.Bool("get", false, `When set, CLI will perform a Get request. Usage: gnmi_cli -get -proto <gnmi.GetRequest> -address <address> [other flags ...]`)
	setFlag          = flag.Bool("set", false, `When set, CLI will perform a Set request. Usage: gnmi_cli -set -proto <gnmi.SetRequest> -address <address> [other flags ...]`)

	withUserPass = flag.Bool("with_user_pass", false, "When set, CLI will prompt for username/password to use when connecting to a target.")

	// Certificate files.
	caCert     = flag.String("ca_crt", "", "CA certificate file. Used to verify server TLS certificate.")
	clientCert = flag.String("client_crt", "", "Client certificate file. Used for client certificate-based authentication.")
	clientKey  = flag.String("client_key", "", "Client private key file. Used for client certificate-based authentication.")
)

func init() {
	flag.Var(clientTypes, "client_types", fmt.Sprintf("List of explicit client types to attempt, one of: %s.", strings.Join(client.RegisteredImpls(), ", ")))
	flag.Var(queryFlag, "query", "Comma separated list of queries.  Each query is a delimited list of OpenConfig path nodes which may also be specified as a glob (*).  The delimeter can be specified with the --delimiter flag.")
	// Query command-line flags.
	flag.Var(queryAddr, "address", "Address of the GNMI target to query.")
	flag.BoolVar(&q.UpdatesOnly, "updates_only", false, "Only stream updates, not the initial sync. Setting this flag for once or polling queries will cause nothing to be returned.")
	// Config command-line flags.
	flag.DurationVar(&cfg.PollingInterval, "polling_interval", 30*time.Second, "Interval at which to poll in seconds if polling is specified for query_type.")
	flag.UintVar(&cfg.Count, "count", 0, "Number of polling/streaming events (0 is infinite).")
	flag.StringVar(&cfg.Delimiter, "delimiter", "/", "Delimiter between path nodes in query. Must be a single UTF-8 code point.")
	flag.DurationVar(&cfg.StreamingDuration, "streaming_duration", 0, "Length of time to collect streaming queries (0 is infinite).")
	flag.StringVar(&cfg.DisplayPrefix, "display_prefix", "", "Per output line prefix.")
	flag.StringVar(&cfg.DisplayIndent, "display_indent", "  ", "Output line, per nesting-level indent.")
	flag.StringVar(&cfg.DisplayType, "display_type", "group", "Display output type (g, group, s, single, p, proto).")
	flag.StringVar(&q.Target, "target", "", "Name of the gNMI target.")
	flag.DurationVar(&q.Timeout, "timeout", 30*time.Second, "Terminate query if no RPC is established within the timeout duration.")
	flag.StringVar(&cfg.Timestamp, "timestamp", "", "Specify timestamp formatting in output.  One of (<empty string>, on, raw, <FORMAT>) where <empty string> is disabled, on is human readable, raw is int64 nanos since epoch, and <FORMAT> is according to golang time.Format(<FORMAT>)")
	flag.BoolVar(&cfg.DisplaySize, "display_size", false, "Display the total size of query response.")
	flag.BoolVar(&cfg.Latency, "latency", false, "Display the latency for receiving each update (Now - update timestamp).")
	flag.StringVar(&q.TLS.ServerName, "server_name", "", "When set, CLI will use this hostname to verify server certificate during TLS handshake.")
	flag.BoolVar(&q.TLS.InsecureSkipVerify, "insecure", false, "When set, CLI will not verify the server certificate during TLS handshake.")

	// Shortcut flags that can be used in place of the longform flags above.
	flag.Var(queryAddr, "a", "Short for address.")
	flag.Var(queryFlag, "q", "Short for query.")
	flag.StringVar(&q.Target, "t", q.Target, "Short for target.")
	flag.BoolVar(&q.UpdatesOnly, "u", q.UpdatesOnly, "Short for updates_only.")
	flag.UintVar(&cfg.Count, "c", cfg.Count, "Short for count.")
	flag.StringVar(&cfg.Delimiter, "d", cfg.Delimiter, "Short for delimiter.")
	flag.StringVar(&cfg.Timestamp, "ts", cfg.Timestamp, "Short for timestamp.")
	flag.StringVar(queryType, "qt", *queryType, "Short for query_type.")
	flag.StringVar(&cfg.DisplayType, "dt", cfg.DisplayType, "Short for display_type.")
	flag.DurationVar(&cfg.StreamingDuration, "sd", cfg.StreamingDuration, "Short for streaming_duration.")
	flag.DurationVar(&cfg.PollingInterval, "pi", cfg.PollingInterval, "Short for polling_interval.")
	flag.BoolVar(&cfg.DisplaySize, "ds", cfg.DisplaySize, "Short for display_size.")
	flag.BoolVar(&cfg.Latency, "l", cfg.Latency, "Short for latency.")
	flag.StringVar(reqProto, "p", *reqProto, "Short for request proto.")
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

	if *caCert != "" {
		certPool := x509.NewCertPool()
		ca, err := ioutil.ReadFile(*caCert)
		if err != nil {
			log.Exitf("could not read %q: %s", *caCert, err)
		}
		if ok := certPool.AppendCertsFromPEM(ca); !ok {
			log.Exit("failed to append CA certificates")
		}

		q.TLS.RootCAs = certPool
	}

	if *clientCert != "" || *clientKey != "" {
		if *clientCert == "" || *clientKey == "" {
			log.Exit("--client_crt and --client_key must be set with file locations")
		}
		certificate, err := tls.LoadX509KeyPair(*clientCert, *clientKey)
		if err != nil {
			log.Exitf("could not load client key pair: %s", err)
		}

		q.TLS.Certificates = []tls.Certificate{certificate}
	}

	var err error
	switch {
	case *capabilitiesFlag: // gnmi.Capabilities
		err = executeCapabilities(ctx)
	case *setFlag: // gnmi.Set
		err = executeSet(ctx)
	case *getFlag: // gnmi.Get
		err = executeGet(ctx)
	default: // gnmi.Subscribe
		err = executeSubscribe(ctx)
	}
	if err != nil {
		log.Error(err)
	}
}

func executeCapabilities(ctx context.Context) error {
	r := &gpb.CapabilityRequest{}
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return fmt.Errorf("unable to parse gnmi.CapabilityRequest from %q : %v", *reqProto, err)
	}
	c, err := gclient.New(ctx, client.Destination{
		Addrs:       q.Addrs,
		Target:      q.Target,
		Timeout:     q.Timeout,
		Credentials: q.Credentials,
		TLS:         q.TLS,
	})
	if err != nil {
		return fmt.Errorf("could not create a gNMI client: %v", err)
	}
	response, err := c.(*gclient.Client).Capabilities(ctx, r)
	if err != nil {
		return fmt.Errorf("target returned RPC error for Capabilities(%q): %v", r.String(), err)
	}
	cfg.Display([]byte(proto.MarshalTextString(response)))
	return nil
}

func executeGet(ctx context.Context) error {
	if *reqProto == "" {
		return errors.New("-proto must be set")
	}
	r := &gpb.GetRequest{}
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return fmt.Errorf("unable to parse gnmi.GetRequest from %q : %v", *reqProto, err)
	}
	c, err := gclient.New(ctx, client.Destination{
		Addrs:       q.Addrs,
		Target:      q.Target,
		Timeout:     q.Timeout,
		Credentials: q.Credentials,
		TLS:         q.TLS,
	})
	if err != nil {
		return fmt.Errorf("could not create a gNMI client: %v", err)
	}
	response, err := c.(*gclient.Client).Get(ctx, r)
	if err != nil {
		return fmt.Errorf("target returned RPC error for Get(%q): %v", r.String(), err)
	}
	cfg.Display([]byte(proto.MarshalTextString(response)))
	return nil
}

func executeSet(ctx context.Context) error {
	if *reqProto == "" {
		return errors.New("-proto must be set")
	}
	r := &gpb.SetRequest{}
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return fmt.Errorf("unable to parse gnmi.SetRequest from %q : %v", *reqProto, err)
	}
	c, err := gclient.New(ctx, client.Destination{
		Addrs:       q.Addrs,
		Target:      q.Target,
		Timeout:     q.Timeout,
		Credentials: q.Credentials,
		TLS:         q.TLS,
	})
	if err != nil {
		return fmt.Errorf("could not create a gNMI client: %v", err)
	}
	response, err := c.(*gclient.Client).Set(ctx, r)
	if err != nil {
		return fmt.Errorf("target returned RPC error for Set(%q) : %v", r, err)
	}
	cfg.Display([]byte(proto.MarshalTextString(response)))
	return nil
}

func executeSubscribe(ctx context.Context) error {
	if *reqProto != "" {
		// Convert SubscribeRequest to a client.Query
		tq, err := cli.ParseSubscribeProto(*reqProto)
		if err != nil {
			log.Exitf("failed to parse gNMI SubscribeRequest text proto: %v", err)
		}
		// Set the fields that are not set as part of conversion above.
		tq.Addrs = q.Addrs
		tq.Credentials = q.Credentials
		tq.Timeout = q.Timeout
		tq.TLS = q.TLS
		return cli.QueryDisplay(ctx, tq, &cfg)
	}

	if q.Type = cli.QueryType(*queryType); q.Type == client.Unknown {
		return errors.New("--query_type must be one of: (o, once, p, polling, s, streaming)")
	}
	// Parse queryFlag into appropriate format.
	if len(*queryFlag) == 0 {
		return errors.New("--query must be set")
	}
	for _, path := range *queryFlag {
		query, err := parseQuery(path, cfg.Delimiter)
		if err != nil {
			return fmt.Errorf("invalid query %q : %v", path, err)
		}
		q.Queries = append(q.Queries, query)
	}
	return cli.QueryDisplay(ctx, q, &cfg)
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

func parseQuery(query, delim string) ([]string, error) {
	d, w := utf8.DecodeRuneInString(delim)
	if w == 0 || w != len(delim) {
		return nil, fmt.Errorf("delimiter must be single UTF-8 codepoint: %q", delim)
	}
	// Ignore leading and trailing delimters.
	query = strings.Trim(query, delim)
	// Split path on delimeter with contextually aware key/value handling.
	var buf []rune
	inKey := false
	null := rune(0)
	for _, r := range query {
		switch r {
		case '[':
			if inKey {
				return nil, fmt.Errorf("malformed query, nested '[': %q ", query)
			}
			inKey = true
		case ']':
			if !inKey {
				return nil, fmt.Errorf("malformed query, unmatched ']': %q", query)
			}
			inKey = false
		case d:
			if !inKey {
				buf = append(buf, null)
				continue
			}
		}
		buf = append(buf, r)
	}
	if inKey {
		return nil, fmt.Errorf("malformed query, missing trailing ']': %q", query)
	}
	return strings.Split(string(buf), string(null)), nil
}
