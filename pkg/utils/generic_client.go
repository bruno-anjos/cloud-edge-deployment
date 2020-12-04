package utils

import (
	"net/http"
)

type GenericClient interface {
	SetHostPort(addr string)
	GetHostPort() string
	GetHTTPClient() *http.Client
}
