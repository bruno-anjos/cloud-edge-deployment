package utils

import (
	"net/http"
	"strconv"
	"time"
)

type Client interface {
	SetHostPort(addr string, port int)
	GetHostPort() string
}

type GenericClient struct {
	hostPort string
	Client   *http.Client
}

const (
	defaultTimeout = 10 * time.Second
)

func NewGenericClient(addr string, port int) *GenericClient {
	return &GenericClient{
		hostPort: addr + ":" + strconv.Itoa(port),
		Client:   &http.Client{Timeout: defaultTimeout},
	}
}

func (c *GenericClient) SetHostPort(addr string, port int) {
	c.hostPort = addr + ":" + strconv.Itoa(port)
}

func (c *GenericClient) GetHostPort() string {
	return c.hostPort
}
