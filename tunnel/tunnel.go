// Package tunnel package wraps grpc_tunnel.
package tunnel

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	log "github.com/golang/glog"

	"github.com/cenkalti/backoff/v4"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc"
	"github.com/openconfig/grpctunnel/tunnel"

	tpb "github.com/openconfig/grpctunnel/proto/tunnel"
)

var (
	// RetryBaseDelay is the initial retry interval for re-connecting tunnel server/client.
	RetryBaseDelay = time.Second
	// RetryMaxDelay caps the retry interval for re-connecting attempts.
	RetryMaxDelay = time.Minute
	// RetryRandomization is the randomization factor applied to the retry
	// interval.
	RetryRandomization = 0.5
)

// ServerConfig defines tunnel server setup.
type ServerConfig struct {
	Addr, CertFile, KeyFile string
}

// Conn is a wraper as a net.Conn interface.
type Conn struct {
	io.ReadWriteCloser
}

// LocalAddr is trivial implementation, in order to match interface net.Conn.
func (tc *Conn) LocalAddr() net.Addr { return nil }

// RemoteAddr is trivial implementation, in order to match interface net.Conn.
func (tc *Conn) RemoteAddr() net.Addr { return nil }

// SetDeadline is trivial implementation, in order to match interface net.Conn.
func (tc *Conn) SetDeadline(t time.Time) error { return nil }

// SetReadDeadline is trivial implementation, in order to match interface net.Conn.
func (tc *Conn) SetReadDeadline(t time.Time) error { return nil }

// SetWriteDeadline is trivial implementation, in order to match interface net.Conn.
func (tc *Conn) SetWriteDeadline(t time.Time) error { return nil }

// Server initiates a tunnel server, and passes all the received targets via a channel.
func Server(ctx context.Context,
	addTargetHandler tunnel.ServerAddTargHandlerFunc,
	delTargetHandler tunnel.ServerDeleteTargHandlerFunc) (*tunnel.Server, error) {

	ts, err := tunnel.NewServer(tunnel.ServerConfig{AddTargetHandler: addTargetHandler, DeleteTargetHandler: delTargetHandler})
	if err != nil {
		return nil, fmt.Errorf("failed to create new server: %v", err)
	}

	return ts, nil
}

// ServerConn (re-)tries and returns a tunnel connection.
func ServerConn(ctx context.Context, ts *tunnel.Server, addr string, target *tunnel.Target) (*Conn, error) {
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 // Retry Subscribe indefinitely.
	bo.InitialInterval = RetryBaseDelay
	bo.MaxInterval = RetryMaxDelay
	bo.RandomizationFactor = RetryRandomization

	for {
		session, err := ts.NewSession(ctx, tunnel.ServerSession{Target: *target})
		if err == nil {
			return &Conn{session}, nil
		}
		duration := bo.NextBackOff()
		time.Sleep(duration)
		log.Infof("Failed to get tunnel connection: %v.\nRetrying in %s.", err, duration)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

	}

}

func startTunnelClient(ctx context.Context, addr string, cert string, chIO chan io.ReadWriteCloser,
	chErr chan error, started chan bool, targets map[tunnel.Target]struct{}) {

	opts := []grpc.DialOption{grpc.WithDefaultCallOptions()}
	if cert == "" {
		opts = append(opts, grpc.WithInsecure())
	} else {
		creds, err := credentials.NewClientTLSFromFile(cert, "")
		if err != nil {
			chErr <- fmt.Errorf("failed to load credentials: %v", err)
			return
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}
	clientConn, err := grpc.Dial(addr, opts...)

	if err != nil {
		chErr <- fmt.Errorf("grpc dial error: %v", err)
		return
	}
	defer clientConn.Close()

	registerHandler := func(t tunnel.Target) error {
		if _, ok := targets[t]; !ok {
			return fmt.Errorf("client cannot handle target ID: %s, type: %s", t.ID, t.Type)
		}
		log.Infof("register handler received id: %v, type: %v", t.ID, t.Type)
		return nil
	}

	handler := func(t tunnel.Target, i io.ReadWriteCloser) error {
		log.Infof("handler called for id: %v, type: %v", t.ID, t.Type)
		chIO <- i
		return nil
	}

	client, err := tunnel.NewClient(tpb.NewTunnelClient(clientConn), tunnel.ClientConfig{
		RegisterHandler: registerHandler,
		Handler:         handler,
	}, targets)
	if err != nil {
		chErr <- fmt.Errorf("failed to create tunnel client: %v", err)
		return
	}

	// Monitor if the client is registered.
	go func() {
		for {
			if client.Registered {
				started <- true
				return
			}
		}
	}()

	if err = client.Run(ctx); err != nil {
		chErr <- err
		return
	}
}

// Listener wraps a tunnel connection.
type Listener struct {
	conn  io.ReadWriteCloser
	addr  tunnelAddr
	chIO  chan io.ReadWriteCloser
	chErr chan error
}

// Accept waits and returns a tunnel connection.
func (l *Listener) Accept() (net.Conn, error) {
	select {
	case err := <-l.chErr:
		return nil, fmt.Errorf("failed to get tunnel listener: %v", err)
	case l.conn = <-l.chIO:
		log.Infof("tunnel listen setup")
		conn := l.conn
		l.conn = nil
		return &Conn{conn}, nil
	}
}

// Close close the embedded connection. Will need more implementation to handle multiple connections.
func (l *Listener) Close() error {
	if l.conn != nil {
		return l.conn.Close()
	}
	return nil
}

// Addr is a trivial implementation.
func (l *Listener) Addr() net.Addr { return l.addr }

type tunnelAddr struct {
	network string
	address string
}

func (a tunnelAddr) Network() string { return a.network }
func (a tunnelAddr) String() string  { return a.address }

// Listen create a tunnel client and returns a Listener.
func Listen(ctx context.Context, addr string, cert string, targets map[tunnel.Target]struct{}) (net.Listener, error) {
	l := Listener{}
	l.addr = tunnelAddr{network: "tcp", address: addr}
	l.chErr = make(chan error)
	l.chIO = make(chan io.ReadWriteCloser)
	started := make(chan bool)

	// Doing a for loop so that it will retry even the tunnel server is not reachable.
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0 // Retry Subscribe indefinitely.
	bo.InitialInterval = RetryBaseDelay
	bo.MaxInterval = RetryMaxDelay
	bo.RandomizationFactor = RetryRandomization

	for {
		go startTunnelClient(ctx, addr, cert, l.chIO, l.chErr, started, targets)

		// tunnel client establishes a tunnel session if it succeeded.
		// retry if it fails.
		select {
		case err := <-l.chErr:
			log.Infof("failed to get tunnel listener: %v", err)
		case <-started:
			log.Infof("tunnel listen setup")
			return &l, nil
		}
		duration := bo.NextBackOff()
		time.Sleep(duration)
		log.Infof("Tunnel listener will retry in %s.", duration)
	}
}
