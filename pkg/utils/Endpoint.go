package utils

import (
	"fmt"
	"net"
	"time"
)

// Endpoint represents the structure for an endpoint
type Endpoint struct {
	Name     string
	Internal *HostPort
	External *HostPort
}

type HostPort struct {
	Address string
	Port    int32
}

type MockOptions struct {
	DesiredEndpoint string
}

func TcpCheck(hp *HostPort, mock *MockOptions) bool {
	if mock != nil {
		return mock.DesiredEndpoint == fmt.Sprintf("%s:%d", hp.Address, hp.Port)
	}

	conn, err := net.DialTimeout("tcp",
		net.JoinHostPort(hp.Address, fmt.Sprint(hp.Port)),
		5*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
