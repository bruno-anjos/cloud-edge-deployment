package client

import (
	"net/http"
	"sync"
	"time"
)

type Client struct {
	hostPort     string
	hostPortLock sync.RWMutex
	Client       *http.Client
}

const (
	defaultTimeout = 10 * time.Second
)

func NewGenericClient(addr string) *Client {
	return &Client{
		hostPort: addr,
		Client: &http.Client{
			Transport:     nil,
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       defaultTimeout,
		},
		hostPortLock: sync.RWMutex{},
	}
}

func (c *Client) SetHostPort(addr string) {
	c.hostPortLock.Lock()
	c.hostPort = addr
	c.hostPortLock.Unlock()
}

func (c *Client) GetHostPort() string {
	c.hostPortLock.RLock()
	defer c.hostPortLock.RUnlock()

	return c.hostPort
}

func (c *Client) GetHTTPClient() *http.Client {
	return c.Client
}
