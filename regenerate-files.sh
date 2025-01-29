#!/bin/bash

# Copyright 2024 Google Inc.
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

BASE=$(bazel info bazel-genfiles)
GNMI_NS='github.com/openconfig/gnmi'


copy_go_generated() {
  pkg="$1"
  # Default to matching all text after the last / in $1 for proto if $2 is unset
  proto="${1##*/}" && [ "${2++}" ] && proto="$2"
  # Bazel go_rules will create empty files containing "// +build ignore\n\npackage ignore"
  # in the case where the protoc compiler doesn't generate any output. See:
  # https://github.com/bazelbuild/rules_go/blob/03a8b8e90eebe699d7/go/tools/builders/protoc.go#L190
  for file in "${BASE}"/"${pkg}"/"${proto}"_go_proto_/"${GNMI_NS}"/"${pkg}"/*.pb.go; do
    [[ $(head -n 1 "${file}") == "// +build ignore" ]] || (cp "${file}" "${pkg}"/ && chmod u+w,-x "${pkg}"/"$(basename "${file}")")
  done
}

bazel build //proto/gnmi_ext:all
copy_go_generated "proto/gnmi_ext"
bazel build //proto/gnmi:all
copy_go_generated "proto/gnmi"
bazel build //testing/fake/proto:all
copy_go_generated "testing/fake/proto" "fake"
bazel build //proto/collector:all
copy_go_generated "proto/collector"
bazel build //proto/target:all
copy_go_generated "proto/target"
