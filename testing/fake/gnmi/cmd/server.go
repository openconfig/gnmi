// server is a simple gRPC gnmi agent implementation which will take a
// configuration and start a listening service for the configured target.
package main

import (
	"fmt"
	"io/ioutil"

	"flag"
	
	log "github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/openconfig/gnmi/testing/fake/gnmi"

	fpb "github.com/openconfig/gnmi/testing/fake/proto"
)

var (
	configFile = flag.String("config", "", "configuration file to load")
	port       = flag.Int("port", -1, "port to listen on")
)

func loadConfig(fileName string) (*fpb.Config, error) {
	in, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	cfg := &fpb.Config{}
	if err := proto.Unmarshal(in, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %v", fileName, err)
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
	cfg.Port = int64(*port)
	a, err := gnmi.New(cfg, nil)
	log.Infof("Starting RPC server on address :%s", a.Address())
	a.Serve() // blocks until close
}
