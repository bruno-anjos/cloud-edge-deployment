package utils

import (
	"net/http"
)

type GenericClient interface {
	GetHTTPClient() *http.Client
}
