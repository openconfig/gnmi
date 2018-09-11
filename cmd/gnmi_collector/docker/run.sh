#!/bin/bash

# Copyright 2017 Google Inc.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This file launches a Docker image of the gNMI collector. It uses
# the configuration stored in the "config" directory (example.cfg)
# and the certificate files stored in this directory. The collector
# listens on tcp/8888 which is forwarded to the host.

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
docker run -d -v $DIR/config:/config -p 8888:8888 openconfig/gnmi_collector
