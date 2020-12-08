package scheduler

import (
	"fmt"
	"net/http"

	"github.com/bruno-anjos/cloud-edge-deployment/api/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
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
	_instanceIdPathVarFormatted = fmt.Sprintf(servers.PathVarFormat, instanceIdPathVar)

	instancesRoute = scheduler.InstancesPath
	instanceRoute  = fmt.Sprintf(scheduler.InstancePath, _instanceIdPathVarFormatted)
)

var Routes = []servers.Route{
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

var DummyRoutes = []servers.Route{
	{
		Name:        startInstanceName,
		Method:      http.MethodPost,
		Pattern:     instancesRoute,
		HandlerFunc: dummyStartInstanceHandler,
	},

	{
		Name:        stopInstanceName,
		Method:      http.MethodDelete,
		Pattern:     instanceRoute,
		HandlerFunc: dummyStopInstanceHandler,
	},

	{
		Name:        stopAllInstancesName,
		Method:      http.MethodDelete,
		Pattern:     instancesRoute,
		HandlerFunc: dummyStopAllInstancesHandler,
	},
}
