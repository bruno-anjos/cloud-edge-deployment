package client

import (
	"net"
	"net/http"
	"time"
)

type Client struct {
	Client *http.Client
}

const (
	defaultTimeout = 10 * time.Second
	maxIdleConns   = 200
)

func NewGenericClient() *Client {
	transport := http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DisableKeepAlives: true,
		DialContext: (&net.Dialer{
			Timeout:   defaultTimeout,
			KeepAlive: defaultTimeout,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          maxIdleConns,
		MaxIdleConnsPerHost:   maxIdleConns,
		MaxConnsPerHost:       maxIdleConns,
	}

	return &Client{
		Client: &http.Client{
			Transport:     &transport,
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       defaultTimeout,
		},
	}
}

func (c *Client) GetHTTPClient() *http.Client {
	return c.Client
}
