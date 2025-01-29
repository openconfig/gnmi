# Copyright 2021 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
"""Dependencies to build gnmi."""

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

def gnmi_deps():
    """Declare the third-party dependencies necessary to build gnmi"""
    if not native.existing_rule("bazel_features"):
        http_archive(
            name = "bazel_features",
            sha256 = "af3d4fb1cf4f25942cb4a933b1ad93a0ea9fe9ee70c2af7f369fb72a67c266e5",
            strip_prefix = "bazel_features-1.21.0",
            url = "https://github.com/bazel-contrib/bazel_features/releases/download/v1.21.0/bazel_features-v1.21.0.tar.gz",
        )
    if not native.existing_rule("bazel_gazelle"):
        http_archive(
            name = "bazel_gazelle",
            sha256 = "aefbf2fc7c7616c9ed73aa3d51c77100724d5b3ce66cfa16406e8c13e87c8b52",
            urls = [
                "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.41.0/bazel-gazelle-v0.41.0.tar.gz",
                "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.41.0/bazel-gazelle-v0.41.0.tar.gz",
            ],
        )
    if not native.existing_rule("com_github_grpc_grpc"):
        http_archive(
            name = "com_github_grpc_grpc",
            url = "https://github.com/grpc/grpc/archive/refs/tags/v1.69.0.tar.gz",
            strip_prefix = "grpc-1.69.0",
            sha256 = "cd256d91781911d46a57506978b3979bfee45d5086a1b6668a3ae19c5e77f8dc",
        )
    if not native.existing_rule("com_google_googleapis"):
        http_archive(
            name = "com_google_googleapis",
            sha256 = "0513f0f40af63bd05dc789cacc334ab6cec27cc89db596557cb2dfe8919463e4",
            strip_prefix = "googleapis-fe8ba054ad4f7eca946c2d14a63c3f07c0b586a0",
            urls = ["https://github.com/googleapis/googleapis/archive/fe8ba054ad4f7eca946c2d14a63c3f07c0b586a0.tar.gz"],
        )
    if not native.existing_rule("com_google_protobuf"):
        http_archive(
            name = "com_google_protobuf",
            url = "https://github.com/protocolbuffers/protobuf/archive/refs/tags/v29.3.zip",
            strip_prefix = "protobuf-29.3",
            sha256 = "85803e01f347141e16a2f770213a496f808fff9f0138c7c0e0c9dfa708b0da92",
            repo_mapping = {
                "@proto_bazel_features": "@bazel_features",
            },
        )
    if not native.existing_rule("bazel_skylib"):
        http_archive(
            name = "bazel_skylib",
            urls = [
                "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.7.1/bazel-skylib-1.7.1.tar.gz",
                "https://github.com/bazelbuild/bazel-skylib/releases/download/1.7.1/bazel-skylib-1.7.1.tar.gz",
            ],
            sha256 = "bc283cdfcd526a52c3201279cda4bc298652efa898b10b4db0837dc51652756f",
        )
    if not native.existing_rule("io_bazel_rules_go"):
        http_archive(
            name = "io_bazel_rules_go",
            sha256 = "0936c9bc3c4321ee372cb8f66dd972d368cb940ed01a9ba9fd7debcf0093f09b",
            urls = [
                "https://github.com/bazelbuild/rules_go/releases/download/v0.51.0/rules_go-v0.51.0.zip",
            ],
        )
    if not native.existing_rule("rules_proto"):
        http_archive(
            name = "rules_proto",
            sha256 = "0e5c64a2599a6e26c6a03d6162242d231ecc0de219534c38cb4402171def21e8",
            strip_prefix = "rules_proto-7.0.2",
            url = "https://github.com/bazelbuild/rules_proto/releases/download/7.0.2/rules_proto-7.0.2.tar.gz",
        )
