#!/bin/bash

# This script is used to generate Go and Python language bindings for the proto files in this repository.
# It uses ghcr.io/srl-labs/protoc container image to generate the bindings. See https://github.com/srl-labs/protoc-container for more details.
# to generate all protos: ./run.sh compile-all
# to generate a specific proto for Go: ./run.sh compile-go-gnmi
# to generate a specific proto for Python: ./run.sh compile-py-gnmi


set -o errexit
set -o pipefail

# container image tag, to see all available tags go to
# https://github.com/srl-labs/protoc-container/pkgs/container/protoc
IMAGE_TAG="22.1__1.28.1"

DOCKER_CMD="docker run -v $(pwd):/in \
  -v $(pwd)/proto:/in/github.com/openconfig/gnmi/proto \
  ghcr.io/srl-labs/protoc:${IMAGE_TAG}"

DOCKER_TERM_CMD="docker run -it -v $(pwd):/in \
  -v $(pwd)/proto:/in/github.com/openconfig/gnmi/proto \
  ghcr.io/srl-labs/protoc:${IMAGE_TAG}"

# import path ".:/usr/include" includes the current working dir (/in) and the installed protobuf .proto files that reside in /usr/include
PROTOC_GO_CMD='protoc -I ".:/protobuf/include" --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative,require_unimplemented_servers=false'

PROTOC_GO_CMD_NOGRPC='protoc -I ".:/protobuf/include" --go_out=. --go_opt=paths=source_relative'

PROTOC_PY_CMD='python3 -m grpc_tools.protoc -I ".:/protobuf/include" --python_out=. --grpc_python_out=.'

function enter-container {
  ${DOCKER_TERM_CMD} ash
}

function compile-all {
  compile-go-gnmi &
  compile-go-collector &
  compile-go-testing-fake &
  compile-go-gnmi-ext &
  compile-go-target &
  
  # python compliations
  compile-py-gnmi &
  compile-py-collector &
  compile-py-testing-fake &
  compile-py-gnmi-ext &
  compile-py-target &

  echo "waiting for all compilations to finish"
  wait
  echo
  echo "All compilations finished!"
}

# generating proto/gnmi/gnmi.proto
function compile-go-gnmi {
    ${DOCKER_CMD} ash -c "${PROTOC_GO_CMD} proto/gnmi/gnmi.proto"
    echo "finished Go compilation for gnmi.proto"
    
}

# generating proto/collector/collector.proto
function compile-go-collector {
    ${DOCKER_CMD} ash -c "${PROTOC_GO_CMD} proto/collector/collector.proto"
    echo "finished Go compilation for collector.proto"

}

# generating testing/fake/proto/fake.proto
function compile-go-testing-fake {
    ${DOCKER_CMD} ash -c "${PROTOC_GO_CMD} testing/fake/proto/fake.proto"
    echo "finished Go compilation for fake.proto"
}

# generating proto/gnmi_ext/gnmi_ext.proto
function compile-go-gnmi-ext {
    ${DOCKER_CMD} ash -c "${PROTOC_GO_CMD_NOGRPC} proto/gnmi_ext/gnmi_ext.proto"
    echo "finished Go compilation for gnmi_ext.proto"
}

# generating proto/target/target.proto
function compile-go-target {
    ${DOCKER_CMD} ash -c "${PROTOC_GO_CMD_NOGRPC} proto/target/target.proto"
    echo "finished Go compilation for target.proto"
}

# generating testing/fake/proto/fake.proto
function compile-py-testing-fake {
    ${DOCKER_CMD} ash -c "${PROTOC_PY_CMD} testing/fake/proto/fake.proto"
    echo "finished Py compilation for fake.proto"
}

# generating proto/gnmi/gnmi.proto
function compile-py-gnmi {
    ${DOCKER_CMD} ash -c "${PROTOC_PY_CMD} proto/gnmi/gnmi.proto"
    echo "finished Py compilation for gnmi.proto"
}

# generating proto/gnmi_ext/gnmi_ext.proto
function compile-py-gnmi-ext {
    ${DOCKER_CMD} ash -c "${PROTOC_PY_CMD} proto/gnmi_ext/gnmi_ext.proto"
    echo "finished Py compilation for gnmi_ext.proto"
}

# generating proto/target/target.proto
function compile-py-target {
    ${DOCKER_CMD} ash -c "${PROTOC_PY_CMD} proto/target/target.proto"
    echo "finished Py compilation for target.proto"
}

# generating proto/collector/collector.proto
function compile-py-collector {
    ${DOCKER_CMD} ash -c "${PROTOC_PY_CMD} proto/collector/collector.proto"
    echo "finished Py compilation for collector.proto"
}


TIMEFORMAT=$'\nTask completed in %3lR'
time "${@:-help}"

set -e