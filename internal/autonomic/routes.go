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
)

// Path variables
const (
	ServiceIdPathVar = "serviceId"
	ChildIdPathVar   = "childId"
)

var (
	_serviceIdPathVarFormatted = fmt.Sprintf(utils.PathVarFormat, ServiceIdPathVar)
	_childIdPathVarFormatted   = fmt.Sprintf(utils.PathVarFormat, ChildIdPathVar)

	servicesRoute     = autonomic.ServicesPath
	serviceRoute      = fmt.Sprintf(autonomic.ServicePath, _serviceIdPathVarFormatted)
	serviceChildRoute = fmt.Sprintf(autonomic.ServiceChildPath, _serviceIdPathVarFormatted, _childIdPathVarFormatted)
)

var Routes = []utils.Route{
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
