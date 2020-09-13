package archimedes

import (
	"strconv"
)

const (
	archimedesServiceName = "archimedes"
	Port                  = 50000
)

var (
	DefaultHostPort = archimedesServiceName + ":" + strconv.Itoa(Port)
)
