# gNMI Collector Docker Image

This directory contains the files required to build a Docker image for the gNMI
collector.

This is not an official Google product.

## Launching the Docker Image

To run the gNMI collector simply execute `run.sh` on a system running the Docker
daemon. By default:

 * The collector listens on tcp/8888 for connections. It can be connected to
   using the gnmi_cli command-line tool.
 * The configuration within the `config` directory (`example.cfg`) is used for
   the collector. This file is a textproto whose format is defined by the
   protobuf in `proto/target/target.proto` within this repository.
 * The SSL certificate (`cert.pem`, `key.pem` within the `config` directory) are
   used for the collector instance. For use of this image anywhere else other
   than for testing, this certificate should be regenerated.

## Building Docker Image

To rebuild the `gnmi_collector` binary, and rebuild the Docker image, execute
the `build.sh` script.

## Image on Docker Hub

This image can be pulled from Docker Hub using:

```
docker pull openconfig/gnmi_collector
```
