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
	"reflect"
	"testing"

	"google.golang.org/grpc/metadata"

	tpb "github.com/openconfig/gnmi/proto/target"
)

func TestGRPCMeta(t *testing.T) {
	tests := []struct {
		name    string
		t       *tpb.Target
		c       CredentialsClient
		want    metadata.MD
		wantErr bool
	}{
		{
			name: "valid password ID",
			t: &tpb.Target{
				Credentials: &tpb.Credentials{
					Username:   "user1",
					PasswordId: "passwordID1",
				},
				Addresses: []string{"111.11.11.111"},
			},
			c: &fakeCreds{},
			want: metadata.Pairs(
				Target, "valid password ID",
				Username, "user1",
				Password, "pass_passwordID1",
			),
		}, {
			name: "valid password",
			t: &tpb.Target{
				Credentials: &tpb.Credentials{
					Username: "user1",
					Password: "password1",
				},
				Addresses: []string{"111.11.11.111"},
			},
			c: &fakeCreds{},
			want: metadata.Pairs(
				Target, "valid password",
				Username, "user1",
				Password, "password1",
			),
		}, {
			name: "username with no password",
			t: &tpb.Target{
				Credentials: &tpb.Credentials{
					Username: "user1",
				},
				Addresses: []string{"111.11.11.111"},
			},
			c:       &fakeCreds{},
			wantErr: true,
		}, {
			name: "valid proxied",
			t: &tpb.Target{
				Credentials: &tpb.Credentials{
					Username:   "user1",
					PasswordId: "passwordID1",
				},
				Addresses: []string{"proxy1;proxy1a;111.11.11.111", "proxy1;proxy1a;222.22.22.222"},
			},
			c: &fakeCreds{},
			want: metadata.Pairs(
				Target, "valid proxied",
				Username, "user1",
				Password, "pass_passwordID1",
				Address, "proxy1a",
				Address, "111.11.11.111",
				Addresses, "proxy1a;111.11.11.111",
				Addresses, "proxy1a;222.22.22.222",
			),
		}, {
			name: "no username",
			t: &tpb.Target{
				Addresses: []string{"111.11.11.111"},
			},
			c: &fakeCreds{},
			want: metadata.Pairs(
				Target, "no username",
			),
		}, {
			name: "credentials fetch error",
			t: &tpb.Target{
				Credentials: &tpb.Credentials{
					Username:   "user1",
					PasswordId: "passwordID1",
				},
				Addresses: []string{"111.11.11.111"},
			},
			c:       &fakeCreds{failNext: true},
			wantErr: true,
		}, {
			name: "replaced meta",
			t: &tpb.Target{
				Credentials: &tpb.Credentials{
					Username:   "user1",
					PasswordId: "passwordID1",
				},
				Addresses: []string{"111.11.11.111"},
				Meta: map[string]string{
					"metaKey1": "foo",
					Password:   "will be replaced",
				},
			},
			c: &fakeCreds{},
			want: metadata.Pairs(
				Target, "replaced meta",
				Username, "user1",
				Password, "pass_passwordID1",
				"metaKey1", "foo",
			),
		},
	}

	for _, tt := range tests {
		got, err := gRPCMeta(context.Background(), tt.name, tt.t, tt.c)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got: %+v, want: %+v", tt.name, got, tt.want)
		}
		switch {
		case err == nil && !tt.wantErr:
		case err == nil && tt.wantErr:
			t.Errorf("%s: got no error, want error.", tt.name)
		case err != nil && !tt.wantErr:
			t.Errorf("%s: got error, want no error.", tt.name)
		}
	}
}
