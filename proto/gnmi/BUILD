load("@com_github_grpc_grpc//bazel:cc_grpc_library.bzl", "cc_grpc_library")
package(
    default_visibility = ["//visibility:public"],
    licenses = ["notice"],
)

filegroup(
    name = "gnmi_proto_srcs",
    srcs = glob(["**/*.proto"]),
    visibility = ["//third_party/pins_infra/thinkit:__pkg__"],
)

proto_library(
    name = "gnmi_proto",
    srcs = ["gnmi.proto"],
    deps = [
        "@com_google_protobuf//:any_proto",
        "@com_google_protobuf//:descriptor_proto",
        "//proto/gnmi_ext:gnmi_ext_proto",
    ],
)

cc_proto_library(
    name = "gnmi_cc_proto",
    deps = [":gnmi_proto"],
)

cc_grpc_library(
    name = "gnmi_grpc_proto",
    srcs = [":gnmi_proto"],
    generate_mocks = True,
    deps = [":gnmi_cc_proto"],
    grpc_only = True,
)

# gNMI should be used with gRPC.
