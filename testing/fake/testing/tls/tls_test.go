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

package tls

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"testing"
)

func TestGenCert(t *testing.T) {
	// Check that generation works at all.
	cert, err := NewCert()
	if err != nil {
		t.Fatal(err)
	}
	// Check that cert is valid by making a dummy connection.
	l, err := tls.Listen("tcp", fmt.Sprint(":0"), &tls.Config{
		Certificates: []tls.Certificate{cert},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() {
		// Just accept the connection to unblock Dial below.
		con, err := l.Accept()
		if err != nil {
			t.Error(err)
		}
		io.Copy(ioutil.Discard, con)
	}()

	con, err := tls.Dial("tcp", l.Addr().String(), &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Error(err)
	}

	if err := con.Handshake(); err != nil {
		t.Error(err)
	}
}
