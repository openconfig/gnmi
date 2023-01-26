/*
Copyright 2018 Google Inc.

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

// The gnmi_collector program implements a caching gNMI collector.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"time"

	
	log "github.com/golang/glog"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc"
	"github.com/openconfig/grpctunnel/dialer"
	"github.com/openconfig/grpctunnel/tunnel"
	"google.golang.org/protobuf/encoding/prototext"
	"github.com/openconfig/gnmi/cache"
	coll "github.com/openconfig/gnmi/collector"
	"github.com/openconfig/gnmi/connection"
	"github.com/openconfig/gnmi/manager"
	"github.com/openconfig/gnmi/subscribe"
	"github.com/openconfig/gnmi/target"

	tunnelpb "github.com/openconfig/grpctunnel/proto/tunnel"
	cpb "github.com/openconfig/gnmi/proto/collector"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
	tpb "github.com/openconfig/gnmi/proto/target"
)

var (
	configFile           = flag.String("config_file", "", "File path for collector configuration.")
	certFile             = flag.String("cert_file", "", "File path for TLS certificate.")
	keyFile              = flag.String("key_file", "", "File path for TLS key.")
	port                 = flag.Int("port", 0, "server port")
	dialTimeout          = flag.Duration("dial_timeout", time.Minute, "Timeout for dialing a connection to a target.")
	metadataUpdatePeriod = flag.Duration("metadata_update_period", 0, "Period for target metadata update. 0 disables updates.")
	sizeUpdatePeriod     = flag.Duration("size_update_period", 0, "Period for updating the target size in metadata. 0 disables updates.")
)

func periodic(period time.Duration, fn func()) {
	if period == 0 {
		return
	}
	t := time.NewTicker(period)
	defer t.Stop()
	for range t.C {
		fn()
	}
}

func (c *collector) addTargetHandler(t tunnel.Target) error {
	log.Infof("tunnel connection attempted from target %s: %s", t.ID, t.Type)

	if t.Type != tunnelpb.TargetType_GNMI_GNOI.String() {
		return fmt.Errorf("received unsupported type type: %s from target: %s, skipping", t.Type, t.ID)
	}

	// We assume the collector's configuration is authoritative on who can connect
	// via tunnels.
	target := c.config.Target[t.ID]
	if target == nil {
		return fmt.Errorf("tunnel target %s not found in config", t.ID)
	}

	if target.GetDialer() == "" {
		return fmt.Errorf("tunnel dialer not specified for target %s", t.ID)
	}

	log.Infof("tunnel target %s accepted", t.ID)
	return nil
}

func (c *collector) deleteTargetHandler(t tunnel.Target) error {
	if t.Type != tunnelpb.TargetType_GNMI_GNOI.String() {
		return fmt.Errorf("received unsupported type type: %s from target: %s, skipping", t.Type, t.ID)
	}
	log.Infof("deleting target %s", t.ID)
	return nil
}

var defaultDialOpts []grpc.DialOption = []grpc.DialOption{
	grpc.WithBlock(),
	grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)),
	// Today, we assume that we should not verify the certificate from the target.
	grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})),
}

// unimplementedCredentials cache credentials for targets.
type unimplementedCredentials struct{}

func (cd *unimplementedCredentials) Lookup(_ context.Context, key string) (string, error) {
	return "", fmt.Errorf("credentials lookup not implemented for key: %q", key)
}

// Under normal conditions, this function will not terminate.  Cancelling
// the context will stop the collector.
func runCollector(ctx context.Context) error {
	if *configFile == "" {
		return errors.New("config_file must be specified")
	}
	if *certFile == "" {
		return errors.New("cert_file must be specified")
	}
	if *keyFile == "" {
		return errors.New("key_file must be specified")
	}

	c := collector{config: &tpb.Configuration{}}

	// Initialize configuration.
	buf, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return fmt.Errorf("Could not read configuration from %q: %v", *configFile, err)
	}
	if err := prototext.Unmarshal(buf, c.config); err != nil {
		return fmt.Errorf("Could not parse configuration from %q: %v", *configFile, err)
	}
	if err := target.Validate(c.config); err != nil {
		return fmt.Errorf("Configuration in %q is invalid: %v", *configFile, err)
	}

	// Initialize TLS credentials.
	creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
	if err != nil {
		return fmt.Errorf("Failed to generate credentials %v", err)
	}

	// Initialize cache.
	c.cache = cache.New(nil)
	// Start functions to periodically update metadata stored in the cache for each target.
	go periodic(*metadataUpdatePeriod, c.cache.UpdateMetadata)
	go periodic(*sizeUpdatePeriod, c.cache.UpdateSize)

	// Initialize tunnel server.
	if c.tServer, err = tunnel.NewServer(tunnel.ServerConfig{AddTargetHandler: c.addTargetHandler, DeleteTargetHandler: c.deleteTargetHandler}); err != nil {
		log.Fatalf("failed to setup tunnel server: %v", err)
	}
	tDialer, err := dialer.FromServer(c.tServer)
	if err != nil {
		log.Fatalf("failed to setup tunnel dialer: %v", err)
	}

	dialerContext := func(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
		return tDialer.DialContext(ctx, target, tunnelpb.TargetType_GNMI_GNOI.String(), opts...)
	}
	// Create connection manager with default dialer. Target specific dialer will be added later.
	c.cm, err = connection.NewManagerCustom(
		map[string]connection.Dial{
			connection.DEFAULT: grpc.DialContext,
			"tunnel":           dialerContext},
		defaultDialOpts...)
	if err != nil {
		return fmt.Errorf("error creating connection.Manager: %v", err)
	}

	c.tm, err = manager.NewManager(manager.Config{
		Credentials:       &unimplementedCredentials{},
		Reset:             c.cache.Reset,
		Sync:              c.cache.Sync,
		Connect:           c.cache.Connect,
		ConnectError:      c.cache.ConnectError,
		ConnectionManager: c.cm,
		Timeout:           *dialTimeout,
		Update: func(target string, v *gnmipb.Notification) {
			// Explicitly set all updates as having come from this target even if
			// the target itself doesn't report or incorrectly reports the target
			// field. Also promote an empty origin field to 'openconfig', the default
			// origin for gNMI.
			if prefix := v.GetPrefix(); prefix == nil {
				v.Prefix = &gnmipb.Path{Origin: "openconfig", Target: target}
			} else {
				if prefix.Origin == "" {
					prefix.Origin = "openconfig"
				}
				prefix.Target = target
			}
			if err := c.cache.GnmiUpdate(v); err != nil {
				if log.V(2) {
					log.Infof("dropped cache.Update(%#v): %v", v, err)
				}
			}
		},
	})
	if err != nil {
		return fmt.Errorf("could not initialize target manager: %v", err)
	}

	// Create a grpc Server.
	srv := grpc.NewServer(grpc.Creds(creds))

	tunnelpb.RegisterTunnelServer(srv, c.tServer)

	// Initialize collectors.
	c.start(context.Background())

	// Initialize the Collector server.
	cpb.RegisterCollectorServer(srv, coll.New(c.tm.Reconnect))
	// Initialize gNMI Proxy Subscribe server.
	subscribeSrv, _ := subscribe.NewServer(c.cache)
	gnmipb.RegisterGNMIServer(srv, subscribeSrv)
	// Forward streaming updates to clients.
	c.cache.SetClient(subscribeSrv.Update)
	// Register listening port and start serving.
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	go srv.Serve(lis)
	defer srv.Stop()
	<-ctx.Done()

	return ctx.Err()
}

// Struct for managing the dynamic creation of tunnel targets.
type tunnTarget struct {
	conf *tpb.Target
	conn *tunnel.Conn
}

type collector struct {
	cache   *cache.Cache
	config  *tpb.Configuration
	tServer *tunnel.Server
	tm      *manager.Manager
	cm      *connection.Manager
}

func (c *collector) add(ctx context.Context, id string, t *tpb.Target, tt *tunnel.Target) error {
	if t == nil {
		return fmt.Errorf("cannot add nil target")
	}

	request, ok := c.config.Request[t.GetRequest()]
	if !ok {
		return fmt.Errorf("no request found for %s", id)
	}

	// Even though address is not meaningful for tunnel target, the manager uses
	// `Addresses` for caching/retrieting connectins. Use the target name itself
	// as the address since target resolution within a tunnel is by target name.
	if t.GetDialer() == "tunnel" {
		t.Addresses = []string{id}
	}

	if err := c.tm.Add(id, t, request); err != nil {
		return fmt.Errorf("Could not add target %q: %v", id, err)
	}
	return nil
}

func (c *collector) start(ctx context.Context) {

	for id, t := range c.config.Target {
		log.Infof("starting target %s", id)

		if err := c.add(ctx, id, t, nil); err != nil {
			log.Errorf("failed to add target %q : %v", id, err)
		}
	}

}

func main() {
	// Flag initialization.
	flag.Parse()
	log.Exit(runCollector(context.Background()))
}
