# This script is used to generate Go and Python language bindings for the proto files in this repository.
# It uses ghcr.io/srl-labs/protoc container image to generate the bindings. See https://github.com/srl-labs/protoc-container for more details.
# to generate all protos: ./compile-protos-docker.sh

# container image tag, to see all available tags go to
# https://github.com/srl-labs/protoc-container/pkgs/container/protoc
IMAGE_TAG="21.12__1.28.1"

DOCKER_CMD="docker run -v $(pwd):/in \
  -v $(pwd)/proto:/in/github.com/openconfig/gnmi/proto \
  ghcr.io/srl-labs/protoc:${IMAGE_TAG}"

PROTOC_GO_CMD='protoc -I ".:/ext_protos" --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative,require_unimplemented_servers=false'

PROTOC_GO_CMD_NOGRPC='protoc -I ".:/ext_protos" --go_out=. --go_opt=paths=source_relative'

#
## Generating Go bindings ##
#

# generating testing/fake/proto/fake.proto
${DOCKER_CMD} \
  bash -c "${PROTOC_GO_CMD} testing/fake/proto/fake.proto"

# generating proto/gnmi/gnmi.proto
${DOCKER_CMD} \
  bash -c "${PROTOC_GO_CMD} proto/gnmi/gnmi.proto"

# generating proto/collector/collector.proto
${DOCKER_CMD} \
  bash -c "${PROTOC_GO_CMD} proto/collector/collector.proto"

# generating proto/gnmi_ext/gnmi_ext.proto
${DOCKER_CMD} \
  bash -c "${PROTOC_GO_CMD_NOGRPC} proto/gnmi_ext/gnmi_ext.proto"

# generating proto/target/target.proto
${DOCKER_CMD} \
  bash -c "${PROTOC_GO_CMD_NOGRPC} proto/target/target.proto"

#
## Generating Python bindings ##
#

PROTOC_PY_CMD='python3 -m grpc_tools.protoc -I ".:/ext_protos" --python_out=. --grpc_python_out=.'

# generating testing/fake/proto/fake.proto
${DOCKER_CMD} \
  bash -c "${PROTOC_PY_CMD} testing/fake/proto/fake.proto"

# generating proto/gnmi/gnmi.proto
${DOCKER_CMD} \
  bash -c "${PROTOC_PY_CMD} proto/gnmi/gnmi.proto"

# generating proto/gnmi_ext/gnmi_ext.proto
${DOCKER_CMD} \
  bash -c "${PROTOC_PY_CMD} proto/gnmi_ext/gnmi_ext.proto"

# generating proto/target/target.proto
${DOCKER_CMD} \
  bash -c "${PROTOC_PY_CMD} proto/target/target.proto"

# generating proto/collector/collector.proto
${DOCKER_CMD} \
  bash -c "${PROTOC_PY_CMD} proto/collector/collector.proto"
