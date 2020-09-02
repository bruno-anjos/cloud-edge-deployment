package autonomic

import (
	"fmt"
	"net/http"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

// Route names
const (
	registerServiceName = "REGISTER_SERVICE"
	deleteServiceName   = "DELETE_SERVICE"
	getAllServicesName  = "GET_ALL_SERVICES"
)

// Path variables
const (
	ServiceIdPathVar = "serviceId"
)

var (
	_serviceIdPathVarFormatted = fmt.Sprintf(utils.PathVarFormat, ServiceIdPathVar)

	servicesRoute = ServicesPath
	serviceRoute  = fmt.Sprintf(ServicePath, _serviceIdPathVarFormatted)
)

var Routes = []utils.Route{
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
