package scheduler

import (
	"fmt"
	"net/http"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

// Route names
const (
	startInstanceName    = "START_INSTANCE"
	stopInstanceName     = "STOP_INSTANCE"
	stopAllInstancesName = "STOP_ALL_INSTANCES"
)

const (
	instanceIdPathVar = "instanceId"
)

var (
	_instanceIdPathVarFormatted = fmt.Sprintf(utils.PathVarFormat, instanceIdPathVar)

	instancesRoute = InstancesPath
	instanceRoute  = fmt.Sprintf(InstancePath, _instanceIdPathVarFormatted)
)

var Routes = []utils.Route{
	{
		Name:        startInstanceName,
		Method:      http.MethodPost,
		Pattern:     instancesRoute,
		HandlerFunc: startInstanceHandler,
	},

	{
		Name:        stopInstanceName,
		Method:      http.MethodDelete,
		Pattern:     instanceRoute,
		HandlerFunc: stopInstanceHandler,
	},

	{
		Name:        stopAllInstancesName,
		Method:      http.MethodDelete,
		Pattern:     instancesRoute,
		HandlerFunc: stopAllInstancesHandler,
	},
}
