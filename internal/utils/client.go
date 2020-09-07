package utils

import (
	"net/http"
	"time"
)

type Client interface {
	SetHostPort(addr string)
	GetHostPort() string
}

type GenericClient struct {
	hostPort string
	Client   *http.Client
}

const (
	defaultTimeout = 10 * time.Second
)

func NewGenericClient(addr string) *GenericClient {
	return &GenericClient{
		hostPort: addr,
		Client:   &http.Client{Timeout: defaultTimeout},
	}
}

func (c *GenericClient) SetHostPort(addr string) {
	c.hostPort = addr
}

func (c *GenericClient) GetHostPort() string {
	return c.hostPort
}
