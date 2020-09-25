package archimedes

import (
	"strconv"
)

const (
	ServiceName = "archimedes"
	Port        = 50000
)

var (
	DefaultHostPort = ServiceName + ":" + strconv.Itoa(Port)
)
