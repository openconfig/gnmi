#!/bin/bash

# Copyright 2018 Google Inc.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This file builds a simple docker image for the gnmi_collector
# based on Alpine Linux.

(cd $GOPATH/src/github.com/openconfig/gnmi/cmd/gnmi_collector && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .)
cp $GOPATH/src/github.com/openconfig/gnmi/cmd/gnmi_collector/gnmi_collector .
docker build -t gnmi_collector -f Dockerfile .
