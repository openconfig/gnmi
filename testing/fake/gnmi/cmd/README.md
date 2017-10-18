## Running the fake gNMI target

First create a config file (text protobuf from `testing/fake/proto/fake.proto`).
A simple example config can be found in `example-config.pb.txt`.

Next, compile the server binary (`go build` or `go install`).

Finally, run:

```
$ ./server --config=example-config.pb.txt --text --port=8080
```

(replace binary and config paths as needed).

Check the `./server --help` output for full flag set. TLS flags are described there.
