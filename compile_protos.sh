#!/bin/sh
set -euo pipefail

proto_imports=".:${GOPATH}/src/github.com/google/protobuf/src:${GOPATH}/src"

# Go
protoc -I=$proto_imports --go_out=. testing/fake/proto/fake.proto
protoc -I=$proto_imports --go_out=plugins=grpc:. proto/gnmi/gnmi.proto

# Python
python -m grpc_tools.protoc -I=$proto_imports --python_out=. --grpc_python_out=. testing/fake/proto/fake.proto
python -m grpc_tools.protoc -I=$proto_imports --python_out=. --grpc_python_out=. proto/gnmi/gnmi.proto
