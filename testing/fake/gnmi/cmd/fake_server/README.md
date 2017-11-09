## Running the fake gNMI target

First create a config file (text protobuf from `testing/fake/proto/fake.proto`).
A simple example config can be found in `example-config.pb.txt`.

Next, compile the server binary (`go build` or `go install`).

Make sure you have TLS key pair for the server (a self-signed pair is fine).

Finally, run:

```
$ ./server --config=example-config.pb.txt --text --port=8080 --logtostderr --server_crt=<path to server TLS cert> --server_key=<path to server TLS key>
```

(replace binary and config paths as needed).

You can skip client authentication with `--allow_no_client_auth`.

Check the `./server --help` output for full flag set. TLS flags are described there.
