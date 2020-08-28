package archimedes

import (
	"strconv"
)

const (
	ArchimedesServiceName = "archimedes"
	Port                  = 50000
)

var (
	DefaultHostPort = ArchimedesServiceName + ":" + strconv.Itoa(Port)
)
