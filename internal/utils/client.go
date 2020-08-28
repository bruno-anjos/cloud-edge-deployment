package utils

import (
	"net/http"
	"strconv"
	"time"
)

type GenericClient struct {
	HostPort string
	Client   *http.Client
}

const (
	defaultTimeout = 10 * time.Second
)

func NewGenericClient(addr string, port int) *GenericClient {
	return &GenericClient{
		HostPort: addr + ":" + strconv.Itoa(port),
		Client:   &http.Client{Timeout: defaultTimeout},
	}
}

func (c *GenericClient) SetHostPort(addr string, port int) {
	c.HostPort = addr + ":" + strconv.Itoa(port)
}
