package utils

import (
	"net/http"
	"sync"
	"time"
)

type Client interface {
	SetHostPort(addr string)
	GetHostPort() string
}

type GenericClient struct {
	hostPort     string
	Client       *http.Client
	hostPortLock sync.RWMutex
}

const (
	defaultTimeout = 10 * time.Second
)

func NewGenericClient(addr string) *GenericClient {
	return &GenericClient{
		hostPort:     addr,
		Client:       &http.Client{Timeout: defaultTimeout},
		hostPortLock: sync.RWMutex{},
	}
}

func (c *GenericClient) SetHostPort(addr string) {
	c.hostPortLock.Lock()
	c.hostPort = addr
	c.hostPortLock.Unlock()
}

func (c *GenericClient) GetHostPort() string {
	c.hostPortLock.RLock()
	defer c.hostPortLock.RUnlock()
	return c.hostPort
}
