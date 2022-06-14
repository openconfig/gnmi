package dialer

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc"
	"github.com/openconfig/grpctunnel/tunnel"
)

func TestNewDialer(t *testing.T) {
	s := &tunnel.Server{}
	tests := []struct {
		desc    string
		s       *tunnel.Server
		wantErr bool
	}{
		{
			desc:    "missing tunnel server",
			wantErr: true,
		}, {
			desc:    "valid",
			s:       s,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		_, err := NewDialer(tt.s)
		switch {
		case err == nil && tt.wantErr:
			t.Errorf("%v: got no error, want error.", tt.desc)
		case err != nil && !tt.wantErr:
			t.Errorf("%v: got error, want no error. err: %v", tt.desc, err)
		}
	}
}

func mockServerConnGood(ctx context.Context, ts *tunnel.Server, target *tunnel.Target) (*tunnel.Conn, error) {
	return &tunnel.Conn{}, nil
}
func mockServerConnBad(ctx context.Context, ts *tunnel.Server, target *tunnel.Target) (*tunnel.Conn, error) {
	return nil, fmt.Errorf("bad tunnel conn")
}

func mockGrpcDialContextGood(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return &grpc.ClientConn{}, nil
}

func mockGrpcDialContextBad(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return nil, fmt.Errorf("bad grpc conn")
}

func TestDial(t *testing.T) {
	s := &tunnel.Server{}
	dialer, err := NewDialer(s)
	if err != nil {
		t.Fatalf("failed to create dialer: %v", err)
	}

	serverConnOrig := serverConn
	grpcDialContextOrig := grpcDialContext
	defer func() {
		serverConn = serverConnOrig
		grpcDialContext = grpcDialContextOrig
	}()

	tests := []struct {
		desc string

		tDial                string
		mockTunnelServerConn func(ctx context.Context, ts *tunnel.Server, target *tunnel.Target) (*tunnel.Conn, error)
		mockGrpcDialContext  func(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
		wantErr              bool
	}{
		{
			desc:                 "succeeded",
			tDial:                "target1",
			mockTunnelServerConn: mockServerConnGood,
			mockGrpcDialContext:  mockGrpcDialContextGood,
			wantErr:              false,
		},
		{
			desc:                 "bad tunnel conn",
			tDial:                "target1",
			mockTunnelServerConn: mockServerConnBad,
			mockGrpcDialContext:  grpcDialContextOrig,
			wantErr:              true,
		},
		{
			desc:                 "bad grpc conn",
			tDial:                "target1",
			mockTunnelServerConn: mockServerConnGood,
			mockGrpcDialContext:  mockGrpcDialContextBad,
			wantErr:              true,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, tt := range tests {
		serverConn = tt.mockTunnelServerConn
		grpcDialContext = tt.mockGrpcDialContext

		_, err := dialer.DialContext(ctx, tt.tDial)
		// Check error.
		if tt.wantErr {
			if err == nil {
				t.Errorf("%v: got no error, want error.", tt.desc)
			}
			continue
		}
		if err != nil {
			t.Errorf("%v: got error, want no error: %v", tt.desc, err)
		}
	}
}
