# gNMI - gRPC Network Management Interface
> Reference Go implementations for gNMI, a service defining an interface for a network management system to interact with a network element.  

**Note:** This is not an official Google product.

The implementations include:

- Client library implementation using `gnmi.proto`
- CLI for interacting with a gNMI service
- Caching collector which connects to multiple gNMI targets and makes the data available over a single gNMI Subscribe RPC to clients

## Client Library
The main entry point for using the client libraries is in
`github.com/openconfig/gnmi/client`.

See [godoc pages](https://godoc.org/github.com/openconfig/gnmi/client) for
documentation and usage examples.

## CLI
The CLI provides reference gNMI functionality for verification against server implementations. For a complete list of arguments, run `gnmi_cli -h`.

```
./gnmi_cli -h
Usage of ./gnmi_cli:
  -a value
    	Short for address.
  -address value
    	Address of the GNMI target to query.
  -alsologtostderr
    	log to standard error as well as files
  -c uint
    	Short for count.
  -ca_crt string
    	CA certificate file. Used to verify server TLS certificate.
  -capabilities
    	When set, CLI will perform a Capabilities request. Usage: gnmi_cli -capabilities [-proto <gnmi.CapabilityRequest>] -address <address> [other flags ...]
  -client_crt string
    	Client certificate file. Used for client certificate-based authentication.
  -client_key string
    	Client private key file. Used for client certificate-based authentication.
  -client_types value
    	List of explicit client types to attempt, one of: gnmi. (default gnmi)
  -count uint
    	Number of polling/streaming events (0 is infinite).
  -credentials_file string
    	File of format { "Username": "demo", "Password": "demo" }
  -d string
    	Short for delimiter. (default "/")
  -delimiter string
    	Delimiter between path nodes in query. Must be a single UTF-8 code point. (default "/")
  -display_indent string
    	Output line, per nesting-level indent. (default "  ")
  -display_prefix string
    	Per output line prefix.
  -display_size
    	Display the total size of query response.
  -display_type string
    	Display output type (g, group, s, single, p, proto). (default "group")
  -ds
    	Short for display_size.
  -dt string
    	Short for display_type. (default "group")
  -get
    	When set, CLI will perform a Get request. Usage: gnmi_cli -get -proto <gnmi.GetRequest> -address <address> [other flags ...]
  -insecure
    	When set, CLI will not verify the server certificate during TLS handshake.
  -insecure_password string
    	Password passed via argument for Username.
  -insecure_username string
    	Username passed via argument.
  -l	Short for latency.
  -latency
    	Display the latency for receiving each update (Now - update timestamp).
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace
  -log_dir string
    	If non-empty, write log files in this directory
  -logtostderr
    	log to standard error instead of files
  -p string
    	Short for request proto.
  -pi duration
    	Short for polling_interval. (default 30s)
  -polling_interval duration
    	Interval at which to poll in seconds if polling is specified for query_type. (default 30s)
  -proto string
    	Text proto for gNMI request.
  -q value
    	Short for query.
  -qt string
    	Short for query_type. (default "once")
  -query value
    	Comma separated list of queries.  Each query is a delimited list of OpenConfig path nodes which may also be specified as a glob (*).  The delimeter can be specified with the --delimiter flag.
  -query_type string
    	Type of result, one of: (o, once, p, polling, s, streaming). (default "once")
  -sd duration
    	Short for streaming_duration.
  -server_name string
    	When set, CLI will use this hostname to verify server certificate during TLS handshake.
  -set
    	When set, CLI will perform a Set request. Usage: gnmi_cli -set -proto <gnmi.SetRequest> -address <address> [other flags ...]
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -streaming_duration duration
    	Length of time to collect streaming queries (0 is infinite).
  -t string
    	Short for target.
  -target string
    	Name of the gNMI target.
  -timeout duration
    	Terminate query if no RPC is established within the timeout duration. (default 30s)
  -timestamp string
    	Specify timestamp formatting in output.  One of (<empty string>, on, raw, <FORMAT>) where <empty string> is disabled, on is human readable, raw is int64 nanos since epoch, and <FORMAT> is according to golang time.Format(<FORMAT>)
  -ts string
    	Short for timestamp.
  -u	Short for updates_only.
  -updates_only
    	Only stream updates, not the initial sync. Setting this flag for once or polling queries will cause nothing to be returned.
  -v value
    	log level for V logs
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
  -with_user_pass
    	When set, CLI will prompt for username/password to use when connecting to a target.
```

### Installation
To install the CLI, run:

```bash
go get -u github.com/openconfig/gnmi/cmd/gnmi_cli
```

### Authentication
There are several methods of authentication, each exclusive in their usage.
* `-with_user_pass` will prompt the user at runtime for authentication credentials.
* `-credentials_file` will expect a JSON file (e.g. `creds.json`) with the `Username` and `Password` fields.

```json
{
    "Username": "demo",
    "Password": "demo"
}
```

* `-insecure_username` and `-insecure_password` expect the associated arguments via the CLI. This is more insecure than other methods and is not generally recommended.

### Proto Specification
The gNMI CLI client can consume a textual proto format which is useful when testing.

`interfaces.proto.txt`
```proto
 subscribe: <
    prefix: <>
    subscription: <
      path: <
            elem: <
                name: "interfaces"
            >
            elem: <
                name: "interface"
            >
        >
      mode: SAMPLE
      sample_interval: 30000000000
    >
    mode: STREAM
    encoding: PROTO
>
```

### Examples

```bash
gnmi_cli -address 127.0.0.1:57400 -insecure -credentials_file creds.json -qt s -proto "$(cat interfaces.proto.txt)"`
```

## Caching Collector
The Caching Collector is a reference collector implementation which may connect to multiple gNMI targets and make the data available over a single gNMI Subscribe RPC to clients. For a complete list of arguments, run `gnmi_collector -h`.

```
./gnmi_collector -h
Usage of ./gnmi_collector:
  -alsologtostderr
    	log to standard error as well as files
  -cert_file string
    	File path for TLS certificate.
  -config_file string
    	File path for collector configuration.
  -dial_timeout duration
    	Timeout for dialing a connection to a target. (default 1m0s)
  -key_file string
    	File path for TLS key.
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace
  -log_dir string
    	If non-empty, write log files in this directory
  -logtostderr
    	log to standard error instead of files
  -metadata_update_period duration
    	Period for target metadata update. 0 disables updates.
  -port int
    	server port
  -size_update_period duration
    	Period for updating the target size in metadata. 0 disables updates.
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -v value
    	log level for V logs
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
```

### Installation
To install the Collector, run:

```bash
go get -u github.com/openconfig/gnmi/cmd/gnmi_collector
```
### Configuration
Configuration is specified with a textual proto format.

`example.cfg`
```proto
request: <
  key: "interfaces"
  value: <
    subscribe: <
      prefix: <
        origin: "openconfig"
      >
      subscription: <
        path: <
          elem: <
            name: "interfaces"
          >
        >
      >
    >
  >
>
target: <
  key: "target1"
  value: <
    address: "127.0.0.1:9998"
    request: "interfaces"
    credentials: <
      username: "username"
      password: "password"
    >
  >
>
```

### Examples
After starting a collector, you should be able to use other gNMI clients to retrieve information from the collector by connecting to the specified port.

```bash
gnmi_collector -config_file example.cfg -cert_file cert.pem -key_file key.pem -port 8888 -alsologtostderr
```
