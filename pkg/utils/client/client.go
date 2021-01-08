package client

import (
	"net/http"
	"time"
)

type Client struct {
	Client *http.Client
}

const (
	defaultTimeout = 10 * time.Second
)

func NewGenericClient() *Client {
	return &Client{
		Client: &http.Client{
			Transport:     nil,
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       defaultTimeout,
		},
	}
}

func (c *Client) GetHTTPClient() *http.Client {
	return c.Client
}
