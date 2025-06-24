# Copyright 2025 Google LLC
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
"""Go dependencies to build gnmi."""

load("@bazel_gazelle//:deps.bzl", "go_repository")

def gnmi_go_deps():
    """Declare the third-party Go dependencies necessary to build gnmi"""
    go_repository(
        name = "co_honnef_go_tools",
        importpath = "honnef.co/go/tools",
        sum = "h1:9MDAWxMoSnB6QoSqiVr7P5mtkT9pOc1kSxchzPCnqJs=",
        version = "v0.4.7",
    )
    go_repository(
        name = "com_github_antihax_optional",
        importpath = "github.com/antihax/optional",
        sum = "h1:xK2lYat7ZLaVVcIuj82J8kIro4V6kDe0AUDFboUCwcg=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_burntsushi_toml",
        importpath = "github.com/BurntSushi/toml",
        sum = "h1:o7IhLm0Msx3BaB+n3Ag7L8EVlByGnpq14C4YWiu/gL8=",
        version = "v1.3.2",
    )
    go_repository(
        name = "com_github_cenkalti_backoff_v4",
        importpath = "github.com/cenkalti/backoff/v4",
        sum = "h1:MyRJ/UdXutAwSAT+s3wNd7MfTIcy71VQueUuFK343L8=",
        version = "v4.3.0",
    )
    go_repository(
        name = "com_github_census_instrumentation_opencensus_proto",
        importpath = "github.com/census-instrumentation/opencensus-proto",
        sum = "h1:glEXhBS5PSLLv4IXzLA5yPRVX4bilULVyxxbrfOtDAk=",
        version = "v0.2.1",
    )
    go_repository(
        name = "com_github_cespare_xxhash",
        importpath = "github.com/cespare/xxhash",
        sum = "h1:a6HrQnmkObjyL+Gs60czilIUGqrzKutQD6XZog3p+ko=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_cespare_xxhash_v2",
        importpath = "github.com/cespare/xxhash/v2",
        sum = "h1:UL815xU9SqsFlibzuggzjXhog7bL6oX9BbNZnL2UFvs=",
        version = "v2.3.0",
    )
    go_repository(
        name = "com_github_client9_misspell",
        importpath = "github.com/client9/misspell",
        sum = "h1:ta993UF76GwbvJcIo3Y68y/M3WxlpEHPWIGDkJYwzJI=",
        version = "v0.3.4",
    )
    go_repository(
        name = "com_github_cncf_udpa_go",
        importpath = "github.com/cncf/udpa/go",
        sum = "h1:cqQfy1jclcSy/FwLjemeg3SR1yaINm74aQyupQ0Bl8M=",
        version = "v0.0.0-20201120205902-5459f2c99403",
    )
    go_repository(
        name = "com_github_cncf_xds_go",
        importpath = "github.com/cncf/xds/go",
        sum = "h1:C5bqEmzEPLsHm9Mv73lSE9e9bKV23aB1vxOsmZrkl3k=",
        version = "v0.0.0-20250326154945-ae57f3c0d45f",
    )
    go_repository(
        name = "com_github_davecgh_go_spew",
        importpath = "github.com/davecgh/go-spew",
        sum = "h1:ZDRjVQ15GmhC3fiQ8ni8+OwkZQO4DARzQgrnXU1Liz8=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_derekparker_trie",
        importpath = "github.com/derekparker/trie",
        sum = "h1:zPGs5GlTifgPVykTSvWB6/+J7EefOmYqH/JuUBI3kf0=",
        version = "v0.0.0-20221221181808-1424fce0c981",
    )
    go_repository(
        name = "com_github_envoyproxy_go_control_plane",
        importpath = "github.com/envoyproxy/go-control-plane",
        sum = "h1:zEqyPVyku6IvWCFwux4x9RxkLOMUL+1vC9xUFv5l2/M=",
        version = "v0.13.4",
    )
    go_repository(
        name = "com_github_envoyproxy_go_control_plane_envoy",
        importpath = "github.com/envoyproxy/go-control-plane/envoy",
        sum = "h1:jb83lalDRZSpPWW2Z7Mck/8kXZ5CQAFYVjQcdVIr83A=",
        version = "v1.32.4",
    )
    go_repository(
        name = "com_github_envoyproxy_go_control_plane_ratelimit",
        importpath = "github.com/envoyproxy/go-control-plane/ratelimit",
        sum = "h1:/G9QYbddjL25KvtKTv3an9lx6VBE2cnb8wp1vEGNYGI=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_envoyproxy_protoc_gen_validate",
        importpath = "github.com/envoyproxy/protoc-gen-validate",
        sum = "h1:DEo3O99U8j4hBFwbJfrz9VtgcDfUKS7KJ7spH3d86P8=",
        version = "v1.2.1",
    )
    go_repository(
        name = "com_github_fsnotify_fsnotify",
        importpath = "github.com/fsnotify/fsnotify",
        sum = "h1:n+5WquG0fcWoWp6xPWfHdbskMCQaFnG6PfBrh1Ky4HY=",
        version = "v1.6.0",
    )
    go_repository(
        name = "com_github_ghodss_yaml",
        importpath = "github.com/ghodss/yaml",
        sum = "h1:wQHKEahhL6wmXdzwWG11gIVCkOv05bNOh+Rxn0yngAk=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_go_jose_go_jose_v4",
        importpath = "github.com/go-jose/go-jose/v4",
        sum = "h1:M6T8+mKZl/+fNNuFHvGIzDz7BTLQPIounk/b9dw3AaE=",
        version = "v4.0.5",
    )
    go_repository(
        name = "com_github_go_logr_logr",
        importpath = "github.com/go-logr/logr",
        sum = "h1:6pFjapn8bFcIbiKo3XT4j/BhANplGihG6tvd+8rYgrY=",
        version = "v1.4.2",
    )
    go_repository(
        name = "com_github_go_logr_stdr",
        importpath = "github.com/go-logr/stdr",
        sum = "h1:hSWxHoqTgW2S2qGc0LTAI563KZ5YKYRhT3MFKZMbjag=",
        version = "v1.2.2",
    )
    go_repository(
        name = "com_github_golang_glog",
        importpath = "github.com/golang/glog",
        sum = "h1:CNNw5U8lSiiBk7druxtSHHTsRWcxKoac6kZKm2peBBc=",
        version = "v1.2.4",
    )
    go_repository(
        name = "com_github_golang_mock",
        importpath = "github.com/golang/mock",
        sum = "h1:G5FRp8JnTd7RQH5kemVNlMeyXQAztQ3mOWV95KxsXH8=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_golang_protobuf",
        importpath = "github.com/golang/protobuf",
        sum = "h1:i7eJL8qZTpSEXOPTxNKhASYpMn+8e5Q6AdndVa1dWek=",
        version = "v1.5.4",
    )
    go_repository(
        name = "com_github_google_go_cmp",
        importpath = "github.com/google/go-cmp",
        sum = "h1:wk8382ETsv4JYUZwIsn6YpYiWiBsYLSJiTsyBybVuN8=",
        version = "v0.7.0",
    )
    go_repository(
        name = "com_github_google_uuid",
        importpath = "github.com/google/uuid",
        sum = "h1:NIvaJDMOsjHA8n1jAhLSgzrAzy1Hgr+hNrb57e+94F0=",
        version = "v1.6.0",
    )
    go_repository(
        name = "com_github_googlecloudplatform_opentelemetry_operations_go_detectors_gcp",
        importpath = "github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp",
        sum = "h1:ErKg/3iS1AKcTkf3yixlZ54f9U1rljCkQyEXWUnIUxc=",
        version = "v1.27.0",
    )
    go_repository(
        name = "com_github_grpc_ecosystem_grpc_gateway",
        importpath = "github.com/grpc-ecosystem/grpc-gateway",
        sum = "h1:gmcG1KaJ57LophUzW0Hy8NmPhnMZb4M0+kPpLofRdBo=",
        version = "v1.16.0",
    )
    go_repository(
        name = "com_github_hashicorp_hcl",
        importpath = "github.com/hashicorp/hcl",
        sum = "h1:0Anlzjpi4vEasTeNFn2mLJgTSwt0+6sfsiTG8qcWGx4=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_inconshreveable_mousetrap",
        importpath = "github.com/inconshreveable/mousetrap",
        sum = "h1:wN+x4NVGpMsO7ErUn/mUI3vEoE6Jt13X2s0bqwp9tc8=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_kr_pretty",
        importpath = "github.com/kr/pretty",
        sum = "h1:flRD4NNwYAUpkphVc1HcthR4KEIFJ65n8Mw5qdRn3LE=",
        version = "v0.3.1",
    )
    go_repository(
        name = "com_github_kr_text",
        importpath = "github.com/kr/text",
        sum = "h1:5Nx0Ya0ZqY2ygV366QzturHI13Jq95ApcVaJBhpS+AY=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_kylelemons_godebug",
        importpath = "github.com/kylelemons/godebug",
        sum = "h1:RPNrshWIDI6G2gRW9EHilWtl7Z6Sb1BR0xunSBf0SNc=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_magiconair_properties",
        importpath = "github.com/magiconair/properties",
        sum = "h1:IeQXZAiQcpL9mgcAe1Nu6cX9LLw6ExEHKjN0VQdvPDY=",
        version = "v1.8.7",
    )
    go_repository(
        name = "com_github_mitchellh_go_wordwrap",
        importpath = "github.com/mitchellh/go-wordwrap",
        sum = "h1:TLuKupo69TCn6TQSyGxwI1EblZZEsQ0vMlAFQflz0v0=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_mitchellh_mapstructure",
        importpath = "github.com/mitchellh/mapstructure",
        sum = "h1:jeMsZIYE/09sWLaz43PL7Gy6RuMjD2eJVyuac5Z2hdY=",
        version = "v1.5.0",
    )
    go_repository(
        name = "com_github_oneofone_xxhash",
        importpath = "github.com/OneOfOne/xxhash",
        sum = "h1:KMrpdQIwFcEqXDklaen+P1axHaj9BSKzvpUUfnHldSE=",
        version = "v1.2.2",
    )
    go_repository(
        name = "com_github_openconfig_goyang",
        importpath = "github.com/openconfig/goyang",
        sum = "h1:JjnPbLY1/y28VyTO67LsEV0TaLWNiZyDcsppGq4F4is=",
        version = "v1.6.0",
    )
    go_repository(
        name = "com_github_openconfig_gribi",
        importpath = "github.com/openconfig/gribi",
        sum = "h1:xMwEg0mBD+21mOxuFOw0d9dBKuIPwJEhMUUeUulZdLg=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_openconfig_grpctunnel",
        build_directives = [
            "gazelle:proto disable",  # keep
            "gazelle:resolve go github.com/openconfig/gnmi/errlist @gnmi//errlist:errlist",  # keep
        ],
        importpath = "github.com/openconfig/grpctunnel",
        sum = "h1:EN99qtlExZczgQgp5ANnHRC/Rs62cAG+Tz2BQ5m/maM=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_openconfig_ygot",
        build_directives = [
            "gazelle:resolve go github.com/openconfig/gnmi/proto/gnmi @gnmi//proto/gnmi:gnmi",  # keep
            "gazelle:resolve go github.com/openconfig/gnmi/errlist @gnmi//errlist:errlist",  # keep
            "gazelle:resolve go github.com/openconfig/gnmi/value @gnmi//value:value",  # keep
        ],
        importpath = "github.com/openconfig/ygot",
        sum = "h1:XHLpwCN91QuKc2LAvnEqtCmH8OuxgLlErDhrdl2mJw8=",
        version = "v0.29.20",
    )
    go_repository(
        name = "com_github_pborman_getopt",
        importpath = "github.com/pborman/getopt",
        sum = "h1:eJ3aFZroQqq0bWmraivjQNt6Dmm5M0h2JcDW38/Azb0=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_pelletier_go_toml_v2",
        importpath = "github.com/pelletier/go-toml/v2",
        sum = "h1:0ctb6s9mE31h0/lhu+J6OPmVeDxJn+kYnJc2jZR9tGQ=",
        version = "v2.0.8",
    )
    go_repository(
        name = "com_github_planetscale_vtprotobuf",
        importpath = "github.com/planetscale/vtprotobuf",
        sum = "h1:GFCKgmp0tecUJ0sJuv4pzYCqS9+RGSn52M3FUwPs+uo=",
        version = "v0.6.1-0.20240319094008-0393e58bdf10",
    )
    go_repository(
        name = "com_github_pmezard_go_difflib",
        importpath = "github.com/pmezard/go-difflib",
        sum = "h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_prometheus_client_model",
        importpath = "github.com/prometheus/client_model",
        sum = "h1:gQz4mCbXsO+nc9n1hCxHcGA3Zx3Eo+UHZoInFGUIXNM=",
        version = "v0.0.0-20190812154241-14fe0d1b01d4",
    )
    go_repository(
        name = "com_github_protocolbuffers_txtpbfmt",
        importpath = "github.com/protocolbuffers/txtpbfmt",
        sum = "h1:ej+64jiny5VETZTqcc1GFVAPEtaSk6U1D0kKC2MS5Yc=",
        version = "v0.0.0-20240823084532-8e6b51fa9bef",
    )
    go_repository(
        name = "com_github_rogpeppe_fastuuid",
        importpath = "github.com/rogpeppe/fastuuid",
        sum = "h1:Ppwyp6VYCF1nvBTXL3trRso7mXMlRrw9ooo375wvi2s=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_rogpeppe_go_internal",
        importpath = "github.com/rogpeppe/go-internal",
        sum = "h1:73kH8U+JUqXU8lRuOHeVHaa/SZPifC7BkcraZVejAe8=",
        version = "v1.9.0",
    )
    go_repository(
        name = "com_github_spaolacci_murmur3",
        importpath = "github.com/spaolacci/murmur3",
        sum = "h1:qLC7fQah7D6K1B0ujays3HV9gkFtllcxhzImRR7ArPQ=",
        version = "v0.0.0-20180118202830-f09979ecbc72",
    )
    go_repository(
        name = "com_github_spf13_afero",
        importpath = "github.com/spf13/afero",
        sum = "h1:stMpOSZFs//0Lv29HduCmli3GUfpFoF3Y1Q/aXj/wVM=",
        version = "v1.9.5",
    )
    go_repository(
        name = "com_github_spf13_cast",
        importpath = "github.com/spf13/cast",
        sum = "h1:R+kOtfhWQE6TVQzY+4D7wJLBgkdVasCEFxSUBYBYIlA=",
        version = "v1.5.1",
    )
    go_repository(
        name = "com_github_spf13_cobra",
        importpath = "github.com/spf13/cobra",
        sum = "h1:7aJaZx1B85qltLMc546zn58BxxfZdR/W22ej9CFoEf0=",
        version = "v1.8.0",
    )
    go_repository(
        name = "com_github_spf13_jwalterweatherman",
        importpath = "github.com/spf13/jwalterweatherman",
        sum = "h1:ue6voC5bR5F8YxI5S67j9i582FU4Qvo2bmqnqMYADFk=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_spf13_pflag",
        importpath = "github.com/spf13/pflag",
        sum = "h1:iy+VFUOCP1a+8yFto/drg2CJ5u0yRoB7fZw3DKv/JXA=",
        version = "v1.0.5",
    )
    go_repository(
        name = "com_github_spf13_viper",
        importpath = "github.com/spf13/viper",
        sum = "h1:js3yy885G8xwJa6iOISGFwd+qlUo5AvyXb7CiihdtiU=",
        version = "v1.15.0",
    )
    go_repository(
        name = "com_github_spiffe_go_spiffe_v2",
        importpath = "github.com/spiffe/go-spiffe/v2",
        sum = "h1:N2I01KCUkv1FAjZXJMwh95KK1ZIQLYbPfhaxw8WS0hE=",
        version = "v2.5.0",
    )
    go_repository(
        name = "com_github_stretchr_objx",
        importpath = "github.com/stretchr/objx",
        sum = "h1:4G4v2dO3VZwixGIRoQ5Lfboy6nUhCyYzaqnIAPPhYs4=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_stretchr_testify",
        importpath = "github.com/stretchr/testify",
        sum = "h1:nOGnQDM7FYENwehXlg/kFVnos3rEvtKTjRvOWSzb6H4=",
        version = "v1.5.1",
    )
    go_repository(
        name = "com_github_subosito_gotenv",
        importpath = "github.com/subosito/gotenv",
        sum = "h1:X1TuBLAMDFbaTAChgCBLu3DU3UPyELpnF2jjJ2cz/S8=",
        version = "v1.4.2",
    )
    go_repository(
        name = "com_github_yuin_goldmark",
        importpath = "github.com/yuin/goldmark",
        sum = "h1:dPmz1Snjq0kmkz159iL7S6WzdahUTHnHB5M56WFVifs=",
        version = "v1.3.5",
    )
    go_repository(
        name = "com_github_zeebo_errs",
        importpath = "github.com/zeebo/errs",
        sum = "h1:XNdoD/RRMKP7HD0UhJnIzUy74ISdGGxURlYG8HSWSfM=",
        version = "v1.4.0",
    )
    go_repository(
        name = "com_google_cloud_go",
        importpath = "cloud.google.com/go",
        sum = "h1:eOI3/cP2VTU6uZLDYAoic+eyzzB9YyGmJ7eIjl8rOPg=",
        version = "v0.34.0",
    )
    go_repository(
        name = "com_google_cloud_go_compute_metadata",
        importpath = "cloud.google.com/go/compute/metadata",
        sum = "h1:A6hENjEsCDtC1k8byVsgwvVcioamEHvZ4j01OwKxG9I=",
        version = "v0.6.0",
    )
    go_repository(
        name = "dev_cel_expr",
        importpath = "cel.dev/expr",
        sum = "h1:wUb94w6OYQS4uXraxo9U+wUAs9jT47Xvl4iPgAwM2ss=",
        version = "v0.23.0",
    )
    go_repository(
        name = "in_gopkg_check_v1",
        importpath = "gopkg.in/check.v1",
        sum = "h1:yhCVgyC4o1eVCa2tZl7eS0r+SDo693bJlVdllGtEeKM=",
        version = "v0.0.0-20161208181325-20d25e280405",
    )
    go_repository(
        name = "in_gopkg_ini_v1",
        importpath = "gopkg.in/ini.v1",
        sum = "h1:Dgnx+6+nfE+IfzjUEISNeydPJh9AXNNsWbGP9KzCsOA=",
        version = "v1.67.0",
    )
    go_repository(
        name = "in_gopkg_yaml_v2",
        importpath = "gopkg.in/yaml.v2",
        sum = "h1:fvjTMHxHEw/mxHbtzPi3JCcKXQRAnQTBRo6YCJSVHKI=",
        version = "v2.2.3",
    )
    go_repository(
        name = "in_gopkg_yaml_v3",
        importpath = "gopkg.in/yaml.v3",
        sum = "h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=",
        version = "v3.0.1",
    )
    go_repository(
        name = "io_opentelemetry_go_auto_sdk",
        importpath = "go.opentelemetry.io/auto/sdk",
        sum = "h1:cH53jehLUN6UFLY71z+NDOiNJqDdPRaXzTel0sJySYA=",
        version = "v1.1.0",
    )
    go_repository(
        name = "io_opentelemetry_go_contrib_detectors_gcp",
        importpath = "go.opentelemetry.io/contrib/detectors/gcp",
        sum = "h1:bGvFt68+KTiAKFlacHW6AhA56GF2rS0bdD3aJYEnmzA=",
        version = "v1.35.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel",
        importpath = "go.opentelemetry.io/otel",
        sum = "h1:xKWKPxrxB6OtMCbmMY021CqC45J+3Onta9MqjhnusiQ=",
        version = "v1.35.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel_metric",
        importpath = "go.opentelemetry.io/otel/metric",
        sum = "h1:0znxYu2SNyuMSQT4Y9WDWej0VpcsxkuklLa4/siN90M=",
        version = "v1.35.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel_sdk",
        importpath = "go.opentelemetry.io/otel/sdk",
        sum = "h1:iPctf8iprVySXSKJffSS79eOjl9pvxV9ZqOWT0QejKY=",
        version = "v1.35.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel_sdk_metric",
        importpath = "go.opentelemetry.io/otel/sdk/metric",
        sum = "h1:1RriWBmCKgkeHEhM7a2uMjMUfP7MsOF5JpUCaEqEI9o=",
        version = "v1.35.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel_trace",
        importpath = "go.opentelemetry.io/otel/trace",
        sum = "h1:dPpEfJu1sDIqruz7BHFG3c7528f6ddfSWfFDVt/xgMs=",
        version = "v1.35.0",
    )
    go_repository(
        name = "io_opentelemetry_go_proto_otlp",
        importpath = "go.opentelemetry.io/proto/otlp",
        sum = "h1:rwOQPCuKAKmwGKq2aVNnYIibI6wnV7EvzgfTCzcdGg8=",
        version = "v0.7.0",
    )
    go_repository(
        name = "org_bitbucket_creachadair_stringset",
        importpath = "bitbucket.org/creachadair/stringset",
        sum = "h1:t1ejQyf8utS4GZV/4fM+1gvYucggZkfhb+tMobDxYOE=",
        version = "v0.0.14",
    )
    go_repository(
        name = "org_golang_google_appengine",
        importpath = "google.golang.org/appengine",
        sum = "h1:/wp5JvzpHIxhs/dumFmF7BXTf3Z+dd4uXta4kVyO508=",
        version = "v1.4.0",
    )
    go_repository(
        name = "org_golang_google_genproto",
        importpath = "google.golang.org/genproto",
        sum = "h1:S9GbmC1iCgvbLyAokVCwiO6tVIrU9Y7c5oMx1V/ki/Y=",
        version = "v0.0.0-20221024183307-1bc688fe9f3e",
    )
    go_repository(
        name = "org_golang_google_genproto_googleapis_api",
        importpath = "google.golang.org/genproto/googleapis/api",
        sum = "h1:hE3bRWtU6uceqlh4fhrSnUyjKHMKB9KrTLLG+bc0ddM=",
        version = "v0.0.0-20250324211829-b45e905df463",
    )
    go_repository(
        name = "org_golang_google_genproto_googleapis_rpc",
        importpath = "google.golang.org/genproto/googleapis/rpc",
        sum = "h1:fc6jSaCT0vBduLYZHYrBBNY4dsWuvgyff9noRNDdBeE=",
        version = "v0.0.0-20250603155806-513f23925822",
    )
    go_repository(
        name = "org_golang_google_grpc",
        importpath = "google.golang.org/grpc",
        sum = "h1:VIWSmpI2MegBtTuFt5/JWy2oXxtjJ/e89Z70ImfD2ok=",
        version = "v1.73.0",
    )
    go_repository(
        name = "org_golang_google_grpc_cmd_protoc_gen_go_grpc",
        importpath = "google.golang.org/grpc/cmd/protoc-gen-go-grpc",
        sum = "h1:F29+wU6Ee6qgu9TddPgooOdaqsxTMunOoj8KA5yuS5A=",
        version = "v1.5.1",
    )
    go_repository(
        name = "org_golang_google_protobuf",
        build_file_proto_mode = "disable_global",  # Manually added to fix build. See https://github.com/golang/protobuf/issues/1611
        importpath = "google.golang.org/protobuf",
        sum = "h1:z1NpPI8ku2WgiWnf+t9wTPsn6eP1L7ksHUlkfLvd9xY=",
        version = "v1.36.6",
    )
    go_repository(
        name = "org_golang_x_crypto",
        importpath = "golang.org/x/crypto",
        sum = "h1:SHs+kF4LP+f+p14esP5jAoDpHU8Gu/v9lFRK6IT5imM=",
        version = "v0.39.0",
    )
    go_repository(
        name = "org_golang_x_exp",
        importpath = "golang.org/x/exp",
        sum = "h1:7dEasQXItcW1xKJ2+gg5VOiBnqWrJc+rq0DPKyvvdbY=",
        version = "v0.0.0-20241009180824-f66d83c29e7c",
    )
    go_repository(
        name = "org_golang_x_exp_typeparams",
        importpath = "golang.org/x/exp/typeparams",
        sum = "h1:UhRVJ0i7bF9n/Hd8YjW3eKjlPVBHzbQdxrBgjbSKl64=",
        version = "v0.0.0-20231219180239-dc181d75b848",
    )
    go_repository(
        name = "org_golang_x_lint",
        importpath = "golang.org/x/lint",
        sum = "h1:VLliZ0d+/avPrXXH+OakdXhpJuEoBZuwh1m2j7U6Iug=",
        version = "v0.0.0-20210508222113-6edffad5e616",
    )
    go_repository(
        name = "org_golang_x_mod",
        importpath = "golang.org/x/mod",
        sum = "h1:n7a+ZbQKQA/Ysbyb0/6IbB1H/X41mKgbhfv7AfG/44w=",
        version = "v0.25.0",
    )
    go_repository(
        name = "org_golang_x_net",
        importpath = "golang.org/x/net",
        sum = "h1:vBTly1HeNPEn3wtREYfy4GZ/NECgw2Cnl+nK6Nz3uvw=",
        version = "v0.41.0",
    )
    go_repository(
        name = "org_golang_x_oauth2",
        importpath = "golang.org/x/oauth2",
        sum = "h1:CrgCKl8PPAVtLnU3c+EDw6x11699EWlsDeWNWKdIOkc=",
        version = "v0.28.0",
    )
    go_repository(
        name = "org_golang_x_sync",
        importpath = "golang.org/x/sync",
        sum = "h1:KWH3jNZsfyT6xfAfKiz6MRNmd46ByHDYaZ7KSkCtdW8=",
        version = "v0.15.0",
    )
    go_repository(
        name = "org_golang_x_sys",
        importpath = "golang.org/x/sys",
        sum = "h1:q3i8TbbEz+JRD9ywIRlyRAQbM0qF7hu24q3teo2hbuw=",
        version = "v0.33.0",
    )
    go_repository(
        name = "org_golang_x_term",
        importpath = "golang.org/x/term",
        sum = "h1:DR4lr0TjUs3epypdhTOkMmuF5CDFJ/8pOnbzMZPQ7bg=",
        version = "v0.32.0",
    )
    go_repository(
        name = "org_golang_x_text",
        importpath = "golang.org/x/text",
        sum = "h1:P42AVeLghgTYr4+xUnTRKDMqpar+PtX7KWuNQL21L8M=",
        version = "v0.26.0",
    )
    go_repository(
        name = "org_golang_x_tools",
        importpath = "golang.org/x/tools",
        sum = "h1:4qz2S3zmRxbGIhDIAgjxvFutSvH5EfnsYrRBj0UI0bc=",
        version = "v0.33.0",
    )
    go_repository(
        name = "org_golang_x_xerrors",
        importpath = "golang.org/x/xerrors",
        sum = "h1:go1bK/D/BFZV2I8cIQd1NKEZ+0owSTG1fDTci4IqFcE=",
        version = "v0.0.0-20200804184101-5ec99f83aff1",
    )
