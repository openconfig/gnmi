## Running the fake gNMI target

First create a config file (text protobuf from `testing/fake/proto/fake.proto`).

You can use a `gen_fake_config` tool in this repo (modify
`../gen_fake_config/gen_config.go` as needed and `go run
../gen_fake_config/gen_config.go`).

Next, compile the server binary (`go build` or `go install`).

Make sure you have TLS key pair for the server (a self-signed pair is fine).

Finally, run:

```
$ fake_server --config config.pb.txt --text --port 8080 --server_crt <path to server TLS cert> --server_key <path to server TLS key> --allow_no_client_auth --logtostderr
```

(replace binary and config paths as needed).

If you wish to verify client TLS authentication, don't use
`--allow_no_client_auth` and set `--ca_crt` to CA certificate path.

To verify your server is running, try:

```
$ gnmi_cli -a localhost:8080 -q '*' -logtostderr -insecure -qt s
```

Check the `./server --help` output for full flag set.
