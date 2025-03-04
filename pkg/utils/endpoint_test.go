package utils

import (
	"net"
	"strconv"
	"testing"
)

func makeTestServer(t *testing.T) (net.Listener, int32) {
	// Start a test TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}

	// Get the dynamic port assigned
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	portInt, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("failed to convert port to int: %v", err)
	}

	// Run test server in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	return listener, int32(portInt)
}

func TestTcpCheck(t *testing.T) {
	listener, port := makeTestServer(t)
	defer listener.Close()

	tests := []struct {
		name string
		hp   *HostPort
		mock *MockOptions
		want bool
	}{
		{
			name: "mock is empty struct ptr",
			hp: &HostPort{
				Address: "127.0.0.1",
				Port:    port,
			},
			mock: &MockOptions{},
			want: false,
		},
		{
			name: "successful connection",
			hp: &HostPort{
				Address: "127.0.0.1",
				Port:    port,
			},
			mock: nil,
			want: true,
		},
		{
			name: "failed connection",
			hp: &HostPort{
				Address: "127.0.0.1",
				Port:    12345, // Using an unlikely to be open port
			},
			mock: nil,
			want: false,
		},
		{
			name: "mock matching endpoint",
			hp: &HostPort{
				Address: "test.example.com",
				Port:    8080,
			},
			mock: &MockOptions{
				DesiredEndpoint: "test.example.com:8080",
			},
			want: true,
		},
		{
			name: "mock non-matching endpoint",
			hp: &HostPort{
				Address: "test.example.com",
				Port:    8080,
			},
			mock: &MockOptions{
				DesiredEndpoint: "other.example.com:8080",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TcpCheck(tt.hp, tt.mock)
			if got != tt.want {
				t.Errorf("TcpCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}
