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

// The gen_fake_config command converts a hardcoded fake.proto message into a
// textual protobuf. The source code is intended to be modified manually to
// generate a valid text proto, instead of writing it by hand.
package main

import (
	"io/ioutil"
	"os"

	log "github.com/golang/glog"
	"github.com/golang/protobuf/proto"

	fpb "github.com/openconfig/gnmi/testing/fake/proto"
)

// Modify the config below to change generated output.
var (
	outputPath = "config.pb.txt"

	config = &fpb.Config{
		Target: "fake target name",
		Seed:   12345,
		Values: []*fpb.Value{
			{
				Path:   []string{"a", "b"},
				Repeat: 3,
				Value:  &fpb.Value_IntValue{&fpb.IntValue{Value: 4}},
			},
			{
				Path:   []string{"b", "c"},
				Repeat: 5,
				Value:  &fpb.Value_StringValue{&fpb.StringValue{Value: "foo"}},
			},
		},
		DisableSync: false,
		DisableEof:  false,
		EnableDelay: false,
		ClientType:  fpb.Config_GRPC_GNMI,
	}
)

func main() {
	out := proto.MarshalTextString(config)
	if err := ioutil.WriteFile(outputPath, []byte(out), os.ModePerm); err != nil {
		log.Exit(err)
	}
}
