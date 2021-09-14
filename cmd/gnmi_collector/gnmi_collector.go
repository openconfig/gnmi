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
	"net"
	"sync"
	"time"

	
	log "github.com/golang/glog"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc"
	"github.com/openconfig/grpctunnel/tunnel"
	"github.com/golang/protobuf/proto"
	"github.com/openconfig/gnmi/cache"
	"github.com/openconfig/gnmi/client"
	gnmiclient "github.com/openconfig/gnmi/client/gnmi"
	coll "github.com/openconfig/gnmi/collector"
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

func (c *collector) isTargetActive(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.tActive[id]
	return ok
}

func (c *collector) addTargetHandler(t tunnel.Target) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.tActive[t.ID]; ok {
		return fmt.Errorf("trying to add target %s, but already in config", t.ID)
	}
	c.chAddTarget <- t
	return nil
}

func (c *collector) deleteTargetHandler(t tunnel.Target) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.tActive[t.ID]; !ok {
		return fmt.Errorf("trying to delete target %s, but not found in config", t.ID)
	}
	c.chDeleteTarget <- t
	return nil
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

	c := collector{config: &tpb.Configuration{},
		cancelFuncs:    map[string]func(){},
		tActive:        map[string]*tunnTarget{},
		chDeleteTarget: make(chan tunnel.Target, 1),
		chAddTarget:    make(chan tunnel.Target, 1),
		addr:           fmt.Sprintf("localhost:%d", *port)}

	// Initialize configuration.
	buf, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return fmt.Errorf("Could not read configuration from %q: %v", *configFile, err)
	}
	if err := proto.UnmarshalText(string(buf), c.config); err != nil {
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

	// Create a grpc Server.
	srv := grpc.NewServer(grpc.Creds(creds))

	// Initialize tunnel server.
	c.tServer, err = tunnel.NewServer(tunnel.ServerConfig{AddTargetHandler: c.addTargetHandler, DeleteTargetHandler: c.deleteTargetHandler})
	if err != nil {
		log.Fatalf("failed to setup tunnel server: %v", err)
	}
	tunnelpb.RegisterTunnelServer(srv, c.tServer)

	// Initialize cache.
	c.cache = cache.New(nil)

	// Start functions to periodically update metadata stored in the cache for each target.
	go periodic(*metadataUpdatePeriod, c.cache.UpdateMetadata)
	go periodic(*sizeUpdatePeriod, c.cache.UpdateSize)

	// Initialize collectors.
	c.start(context.Background())

	// Initialize the Collector server.
	cpb.RegisterCollectorServer(srv, coll.New(c.reconnect))
	// Initialize gNMI Proxy Subscribe server.
	subscribeSrv, err := subscribe.NewServer(c.cache)
	if err != nil {
		return fmt.Errorf("Could not instantiate gNMI server: %v", err)
	}
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

// Container for some of the target state data. It is created once
// for every device and used as a closure parameter by ProtoHandler.
type state struct {
	name   string
	target *cache.Target
	// connected status is set to true when the first gnmi notification is received.
	// it gets reset to false when disconnect call back of ReconnectClient is called.
	connected bool
}

func (s *state) disconnect() {
	s.connected = false
	s.target.Reset()
}

// handleUpdate parses a protobuf message received from the target. This implementation handles only
// gNMI SubscribeResponse messages. When the message is an Update, the GnmiUpdate method of the
// cache.Target is called to generate an update. If the message is a sync_response, then target is
// marked as synchronised.
func (s *state) handleUpdate(msg proto.Message) error {
	if !s.connected {
		s.target.Connect()
		s.connected = true
	}
	resp, ok := msg.(*gnmipb.SubscribeResponse)
	if !ok {
		return fmt.Errorf("failed to type assert message %#v", msg)
	}
	switch v := resp.Response.(type) {
	case *gnmipb.SubscribeResponse_Update:
		// Gracefully handle gNMI implementations that do not set Prefix.Target in their
		// SubscribeResponse Updates.
		if v.Update.GetPrefix() == nil {
			v.Update.Prefix = &gnmipb.Path{}
		}
		if v.Update.Prefix.Target == "" {
			v.Update.Prefix.Target = s.name
		}
		s.target.GnmiUpdate(v.Update)
	case *gnmipb.SubscribeResponse_SyncResponse:
		s.target.Sync()
	case *gnmipb.SubscribeResponse_Error:
		return fmt.Errorf("error in response: %s", v)
	default:
		return fmt.Errorf("unknown response %T: %s", v, v)
	}
	return nil
}

// Struct for managing the dynamic creation of tunnel targets.
type tunnTarget struct {
	conf *tpb.Target
	conn *tunnel.Conn
}

type collector struct {
	cache          *cache.Cache
	config         *tpb.Configuration
	mu             sync.Mutex
	cancelFuncs    map[string]func()
	addr           string
	tActive        map[string]*tunnTarget
	tServer        *tunnel.Server
	tRequest       string
	chAddTarget    chan tunnel.Target
	chDeleteTarget chan tunnel.Target
}

func (c *collector) addCancel(target string, cancel func()) {
	c.mu.Lock()
	c.cancelFuncs[target] = cancel
	c.mu.Unlock()
}

func (c *collector) reconnect(target string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	cancel, ok := c.cancelFuncs[target]
	if !ok {
		return fmt.Errorf("no such target: %q", target)
	}
	cancel()
	delete(c.cancelFuncs, target)
	return nil
}

func (c *collector) runSingleTarget(ctx context.Context, targetID string, tc *tunnel.Conn) {
	c.mu.Lock()
	target, ok := c.tActive[targetID]
	c.mu.Unlock()
	if !ok {
		log.Errorf("Unknown tunnel target %q", targetID)
		return
	}

	go func(name string, target *tunnTarget) {
		s := &state{name: name, target: c.cache.Add(name)}
		qr := c.config.Request[target.conf.GetRequest()]
		q, err := client.NewQuery(qr)
		if err != nil {
			log.Errorf("NewQuery(%s): %v", qr.String(), err)
			return
		}
		q.Addrs = target.conf.GetAddresses()

		if target.conf.Credentials != nil {
			q.Credentials = &client.Credentials{
				Username: target.conf.Credentials.Username,
				Password: target.conf.Credentials.Password,
			}
		}

		// TLS is always enabled for a target.
		q.TLS = &tls.Config{
			// Today, we assume that we should not verify the certificate from the target.
			InsecureSkipVerify: true,
		}

		q.Target = name
		q.Timeout = *dialTimeout
		q.ProtoHandler = s.handleUpdate
		q.TunnelConn = target.conn
		if err := q.Validate(); err != nil {
			log.Errorf("query.Validate(): %v", err)
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
		cl := client.BaseClient{}
		if err := cl.Subscribe(ctx, q, gnmiclient.Type); err != nil {
			log.Errorf("Subscribe failed for target %q: %v", name, err)
			// remove target once it becomes unavailable
			c.removeTarget(name)
		}
	}(targetID, target)
}

func (c *collector) start(ctx context.Context) {
	// First, run for non-tunnel targets.
	useTunnel := false
	for id, t := range c.config.Target {
		if t.GetDialer() == "tunnel" {
			useTunnel = true
			continue
		}
		log.Infof("adding non-tunnel target %s", id)
		c.addTarget(ctx, id, &tunnTarget{conf: t})
		c.runSingleTarget(ctx, id, nil)
	}

	// If no tunnels are configured then return.
	if !useTunnel {
		return
	}

	// Monitor targets from the tunnel.
	go func() {
		log.Info("watching for tunnel connections")
		for {
			select {
			case target := <-c.chAddTarget:
				log.Infof("adding target: %+v", target)
				if target.Type != tunnelpb.TargetType_GNMI_GNOI.String() {
					log.Infof("received unsupported type type: %s from target: %s, skipping", target.Type, target.ID)
					continue
				}
				if c.isTargetActive(target.ID) {
					log.Infof("received target %s, which is already registered. skipping", target.ID)
					continue
				}

				ctx, cancel := context.WithCancel(ctx)
				c.addCancel(target.ID, cancel)

				tc, err := tunnel.ServerConn(ctx, c.tServer, c.addr, &target)
				if err != nil {
					log.Errorf("failed to get new tunnel session for target %v:%v", target.ID, err)
					continue
				}
				cfg := c.config.Target[target.ID]
				if cfg == nil {
					cfg = &tpb.Target{
						Request: c.tRequest,
					}
				}
				t := &tunnTarget{
					conn: tc,
					conf: cfg,
				}
				c.addTarget(ctx, target.ID, t)
				c.runSingleTarget(ctx, target.ID, tc)

			case target := <-c.chDeleteTarget:
				log.Infof("deleting target: %+v", target)
				if target.Type != tunnelpb.TargetType_GNMI_GNOI.String() {
					log.Infof("received unsupported type type: %s from target: %s, skipping", target.Type, target.ID)
					continue
				}
				if !c.isTargetActive(target.ID) {
					log.Infof("received target %s, which is not registered. skipping", target.ID)
					continue
				}
				c.removeTarget(target.ID)
			}
		}
	}()
}

func (c *collector) removeTarget(target string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	t, ok := c.tActive[target]
	if !ok {
		log.Infof("trying to remove target %s, but not found in config. do nothing", target)
		return nil
	}
	delete(c.tActive, target)
	cancel, ok := c.cancelFuncs[target]
	if !ok {
		return fmt.Errorf("cannot find cancel for target %q", target)
	}
	cancel()
	delete(c.cancelFuncs, target)
	t.conn.Close()
	log.Infof("target %s removed", target)
	return nil
}

func (c *collector) addTarget(ctx context.Context, name string, target *tunnTarget) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.tActive[name]; ok {
		log.Infof("trying to add target %q:%+v, but already in exists. do nothing", name, target)
		return nil
	}

	// For non-tunnel target, don't modify the addresses.
	if target.conf.GetDialer() == "tunnel" {
		target.conf.Addresses = []string{c.addr}
	}

	c.tActive[name] = target
	log.Infof("Added target: %q:%+v", name, target)
	return nil
}

func main() {
	// Flag initialization.
	flag.Parse()
	log.Exit(runCollector(context.Background()))
}
