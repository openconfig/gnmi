/*
Copyright 2020 Google Inc.

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

package manager

import (
	"context"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	"google.golang.org/grpc/metadata"

	tpb "github.com/openconfig/gnmi/proto/target"
)

// gRPC metadata keys.
const (
	// Address is deprecated, proxies should support multiple addresses via the
	// Addresses metadata key.
	Address   = "address"
	Addresses = "addresses"
	Password  = "password"
	Target    = "target"
	Username  = "username"
)

func addrChains(addrs []string) [][]string {
	ac := make([][]string, len(addrs))
	for idx, addrLine := range addrs {
		ac[idx] = strings.Split(addrLine, AddrSeparator)
	}
	return ac
}

func gRPCMeta(ctx context.Context, name string, t *tpb.Target, cred CredentialsClient) (metadata.MD, error) {
	meta := metadata.MD{}
	c := t.GetCredentials()
	if user := c.GetUsername(); user != "" {
		if id := c.GetPasswordId(); id != "" {
			if cred == nil {
				return nil, fmt.Errorf("nil CredentialsClient used to lookup password ID for target %q", name)
			}
			p, err := cred.Lookup(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("failed loading credentials for target %q: %v", name, err)
			}
			if p != "" {
				meta.Set(Password, p)
			}
		} else if p := c.GetPassword(); p != "" {
			meta.Set(Password, p)
		} else {
			return nil, fmt.Errorf("username has no associated password/password ID for target %q", name)
		}
		meta.Set(Username, user)
	}
	if name != "" {
		meta.Set(Target, name)
	}
	for idx, ac := range addrChains(t.GetAddresses()) {
		if len(ac) == 0 {
			return nil, fmt.Errorf("empty address chain for target %q", name)
		}
		if idx == 0 {
			meta.Set(Address, ac[1:]...)
		}
		next := strings.Join(ac[1:], AddrSeparator)
		if next == "" {
			continue
		}
		meta.Append(Addresses, next)
	}
	for k, v := range t.GetMeta() {
		if _, ok := meta[k]; ok {
			log.Warningf("Prefer generated metadata key over existing: %v", k)
			continue
		}
		meta.Set(k, v)
	}
	return meta, nil
}
