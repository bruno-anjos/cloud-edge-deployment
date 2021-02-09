package utils

import (
	log "github.com/sirupsen/logrus"
)

const (
	errorHTTPClietNilFormat = "httpclient is nil"
)

func PanicOnErrFromChan(errChan <-chan error) {
	log.Panic(<-errChan)
}
