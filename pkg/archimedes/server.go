package archimedes

import (
	"strconv"

	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

const (
	Port = 50000
)

var (
	DefaultHostPort = publicUtils.ArchimedesServiceName + ":" + strconv.Itoa(Port)
)
