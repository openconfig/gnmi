# gNMI - gRPC Network Management Interface

This repository contains reference Go implementations for gNMI.

**Note:** This is not an official Google product.

The implementations include:

- client library implementation using `gnmi.proto`
- CLI for interacting with a gNMI service
- Caching collector that connects to multiple gNMI targets and makes the data
  available over a single gNMI Subscribe RPC to clients

## Installing

To install the CLI, run

    go get -u github.com/openconfig/gnmi/cmd/gnmi_cli

## Client libraries

The main entry point for using the client libraries is in
`github.com/openconfig/gnmi/client`.

See [godoc pages](https://godoc.org/github.com/openconfig/gnmi/client) for
documentation and usage examples.
