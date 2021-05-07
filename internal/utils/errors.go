package utils

import (
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	errorHTTPClietNilFormat = "httpclient is nil"
	ErrTimedOut             = "timed out"
)

func RetryFuncOnErr(errChan <-chan error, retryFunc func()) {
	log.Warn(<-errChan)
	retryFunc()
}

func PanicOnErrFromChan(errChan <-chan error) {
	for err := range errChan {
		if !strings.Contains(err.Error(), ErrTimedOut) {
			log.Panic(err)
		}
	}
}
