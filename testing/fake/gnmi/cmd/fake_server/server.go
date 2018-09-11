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

// The fake_server is a simple gRPC gnmi agent implementation which will take a
// configuration and start a listening service for the configured target.
package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"flag"
	
	log "github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc"
	"github.com/openconfig/gnmi/testing/fake/gnmi"

	fpb "github.com/openconfig/gnmi/testing/fake/proto"
)

var (
	configFile = flag.String("config", "", "configuration file to load")
	text       = flag.Bool("text", false, "use text configuration file")
	port       = flag.Int("port", -1, "port to listen on")
	// Certificate files.
	caCert            = flag.String("ca_crt", "", "CA certificate for client certificate validation. Optional.")
	serverCert        = flag.String("server_crt", "", "TLS server certificate")
	serverKey         = flag.String("server_key", "", "TLS server private key")
	allowNoClientCert = flag.Bool("allow_no_client_auth", false, "When set, fake_server will request but not require a client certificate.")
)

func loadConfig(fileName string) (*fpb.Config, error) {
	in, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	cfg := &fpb.Config{}
	if *text {
		if err := proto.UnmarshalText(string(in), cfg); err != nil {
			return nil, fmt.Errorf("failed to parse text file %s: %v", fileName, err)
		}
	} else {
		if err := proto.Unmarshal(in, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %v", fileName, err)
		}
	}
	return cfg, nil
}

func main() {
	flag.Parse()
	switch {
	case *configFile == "":
		log.Errorf("config must be set.")
		return
	case *port < 0:
		log.Errorf("port must be >= 0.")
		return
	}
	cfg, err := loadConfig(*configFile)
	if err != nil {
		log.Errorf("Failed to load %s: %v", *configFile, err)
		return
	}

	certificate, err := tls.LoadX509KeyPair(*serverCert, *serverKey)
	if err != nil {
		log.Exitf("could not load server key pair: %s", err)
	}
	tlsCfg := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
	}
	if *allowNoClientCert {
		// RequestClientCert will ask client for a certificate but won't
		// require it to proceed. If certificate is provided, it will be
		// verified.
		tlsCfg.ClientAuth = tls.RequestClientCert
	}

	if *caCert != "" {
		ca, err := ioutil.ReadFile(*caCert)
		if err != nil {
			log.Exitf("could not read CA certificate: %s", err)
		}
		certPool := x509.NewCertPool()
		if ok := certPool.AppendCertsFromPEM(ca); !ok {
			log.Exit("failed to append CA certificate")
		}
		tlsCfg.ClientCAs = certPool
	}

	opts := []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsCfg))}
	cfg.Port = int32(*port)
	a, err := gnmi.New(cfg, opts)
	if err != nil {
		log.Errorf("Failed to create gNMI server: %v", err)
		return
	}

	log.Infof("Starting RPC server on address: %s", a.Address())
	a.Serve() // blocks until close
}
