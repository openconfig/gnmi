/*
Copyright 2017 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package config provides gRPC configuration methods used by tests to
// facilitate easier setup of gRPC clients and servers.
package config

import (
	"crypto/tls"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc"

	gtls "github.com/openconfig/gnmi/testing/fake/testing/tls"
)

// WithSelfTLSCert generates a new self-signed in-memory TLS certificate and
// returns a grpc.ServerOption containing it. This is only for use in tests.
func WithSelfTLSCert() (grpc.ServerOption, error) {
	cert, err := gtls.NewCert()
	if err != nil {
		return nil, err
	}
	return grpc.Creds(credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
	})), nil
}
