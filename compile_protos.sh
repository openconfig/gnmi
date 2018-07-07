#!/bin/sh

# Copyright 2017 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

proto_imports=".:proto:${GOPATH}/src/github.com/google/protobuf/src:${GOPATH}/src"

out=${GOPATH}/src

# Go
protoc -I=$proto_imports --go_out=$out              proto/gnmi_ext/gnmi_ext.proto
protoc -I=$proto_imports --go_out=plugins=grpc:$out proto/gnmi/gnmi.proto
protoc -I=$proto_imports --go_out=$out              proto/target/target.proto
protoc -I=$proto_imports --go_out=plugins=grpc:$out testing/fake/proto/fake.proto

# Python
python -m grpc_tools.protoc -I=$proto_imports --python_out=.                     proto/gnmi_ext/gnmi_ext.proto
python -m grpc_tools.protoc -I=$proto_imports --python_out=. --grpc_python_out=. proto/gnmi/gnmi.proto
python -m grpc_tools.protoc -I=$proto_imports --python_out=.                     proto/target/target.proto
python -m grpc_tools.protoc -I=$proto_imports --python_out=. --grpc_python_out=. testing/fake/proto/fake.proto
