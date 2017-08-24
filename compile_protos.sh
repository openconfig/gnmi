#!/bin/sh

# Go
protoc -I=testing/fake/proto:${GOPATH}/src --go_out=testing/fake/proto testing/fake/proto/fake.proto
protoc -I=proto/gnmi:${GOPATH}/src --go_out=plugins=grpc:proto/gnmi --proto_path=${GOPATH}/src proto/gnmi/gnmi.proto

# Python
python -m grpc_tools.protoc -I=testing/fake/proto:${GOPATH}/src --python_out=testing/fake/proto --grpc_python_out=testing/fake/proto testing/fake/proto/fake.proto
python -m grpc_tools.protoc -I=proto/gnmi:${GOPATH}/src --python_out=proto/gnmi --grpc_python_out=proto/gnmi proto/gnmi/gnmi.proto
