// server is a simple gRPC gnmi agent implementation which will take a
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
	caCert     = flag.String("ca_crt", "", "CA certificate")
	serverCert = flag.String("server_crt", "", "server certificate")
	serverKey  = flag.String("server_key", "", "server private key")
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

	var opts []grpc.ServerOption
	if *caCert != "" || *serverCert != "" || *serverKey != "" {
		if *caCert == "" || *serverCert == "" || *serverKey == "" {
			log.Exit("--ca_crt --server_crt and --server_key must be set with file locations")
		}

		certificate, err := tls.LoadX509KeyPair(*serverCert, *serverKey)
		if err != nil {
			log.Exitf("could not load server key pair: %s", err)
		}

		certPool := x509.NewCertPool()
		ca, err := ioutil.ReadFile(*caCert)
		if err != nil {
			log.Exitf("could not read CA certificate: %s", err)
		}

		if ok := certPool.AppendCertsFromPEM(ca); !ok {
			log.Exit("failed to append CA certificate")
		}

		creds := credentials.NewTLS(&tls.Config{
			ClientAuth:   tls.RequireAndVerifyClientCert,
			Certificates: []tls.Certificate{certificate},
			ClientCAs:    certPool,
		})

		opts = append(opts, grpc.Creds(creds))
	}

	cfg.Port = int64(*port)
	a, err := gnmi.New(cfg, opts)
	if err != nil {
		log.Errorf("Failed to create gNMI server: %v", err)
		return
	}

	log.Infof("Starting RPC server on address :%s", a.Address())
	a.Serve() // blocks until close
}
