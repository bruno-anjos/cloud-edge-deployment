package autonomic

import (
	"fmt"
	"net/http"

	"github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

// Route names
const (
	registerServiceName    = "REGISTER_SERVICE"
	deleteServiceName      = "DELETE_SERVICE"
	getAllServicesName     = "GET_ALL_SERVICES"
	addServiceChildName    = "ADD_SERVICE_CHILD"
	removeServiceChildName = "REMOVE_SERVICE_CHILD"
	setServiceParentName   = "SET_SERVICE_PARENT"
	isNodeInVicinityName   = "IS_NODE_IN_VICINITY"
	closestNodeName        = "CLOSEST_NODE"
	getVicinityName        = "GET_VICINITY"
	getMyLocationName      = "GET_MY_LOCATION"
	getLoadName            = "GET_LOAD"
)

// Path variables
const (
	serviceIdPathVar = "serviceId"
	childIdPathVar   = "childId"
	parentIdPathVar  = "parentId"
	nodeIdPathVar    = "nodeId"
)

var (
	_serviceIdPathVarFormatted = fmt.Sprintf(utils.PathVarFormat, serviceIdPathVar)
	_childIdPathVarFormatted   = fmt.Sprintf(utils.PathVarFormat, childIdPathVar)
	_parentIdPathVarFormatted  = fmt.Sprintf(utils.PathVarFormat, parentIdPathVar)
	_nodeIdPathVarFormatted    = fmt.Sprintf(utils.PathVarFormat, nodeIdPathVar)

	servicesRoute         = autonomic.ServicesPath
	serviceRoute          = fmt.Sprintf(autonomic.ServicePath, _serviceIdPathVarFormatted)
	serviceChildRoute     = fmt.Sprintf(autonomic.ServiceChildPath, _serviceIdPathVarFormatted, _childIdPathVarFormatted)
	serviceParentRoute    = fmt.Sprintf(autonomic.ServiceParentPath, _serviceIdPathVarFormatted, _parentIdPathVarFormatted)
	isNodeInVicinityRoute = fmt.Sprintf(autonomic.IsNodeInVicinityPath, _nodeIdPathVarFormatted)
	closestNodeRoute      = autonomic.ClosestNodePath
	getVicinityRoute      = autonomic.VicinityPath
	getMyLocationRoute    = autonomic.MyLocationPath
	getLoadRoute          = fmt.Sprintf(autonomic.LoadPath, _serviceIdPathVarFormatted)
)

var Routes = []utils.Route{
	{
		Name:        getLoadName,
		Method:      http.MethodGet,
		Pattern:     getLoadRoute,
		HandlerFunc: getLoadForServiceHandler,
	},

	{
		Name:        getMyLocationName,
		Method:      http.MethodGet,
		Pattern:     getMyLocationRoute,
		HandlerFunc: getMyLocationHandler,
	},

	{
		Name:        getVicinityName,
		Method:      http.MethodGet,
		Pattern:     getVicinityRoute,
		HandlerFunc: getVicinityHandler,
	},

	{
		Name:        closestNodeName,
		Method:      http.MethodGet,
		Pattern:     closestNodeRoute,
		HandlerFunc: closestNodeToHandler,
	},

	{
		Name:        isNodeInVicinityName,
		Method:      http.MethodGet,
		Pattern:     isNodeInVicinityRoute,
		HandlerFunc: isNodeInVicinityHandler,
	},

	{
		Name:        setServiceParentName,
		Method:      http.MethodPost,
		Pattern:     serviceParentRoute,
		HandlerFunc: setServiceParentHandler,
	},

	{
		Name:        addServiceChildName,
		Method:      http.MethodPost,
		Pattern:     serviceChildRoute,
		HandlerFunc: addServiceChildHandler,
	},

	{
		Name:        removeServiceChildName,
		Method:      http.MethodDelete,
		Pattern:     serviceChildRoute,
		HandlerFunc: removeServiceChildHandler,
	},

	{
		Name:        registerServiceName,
		Method:      http.MethodPost,
		Pattern:     serviceRoute,
		HandlerFunc: addServiceHandler,
	},

	{
		Name:        deleteServiceName,
		Method:      http.MethodDelete,
		Pattern:     serviceRoute,
		HandlerFunc: removeServiceHandler,
	},

	{
		Name:        getAllServicesName,
		Method:      http.MethodGet,
		Pattern:     servicesRoute,
		HandlerFunc: getAllServicesHandler,
	},
}
