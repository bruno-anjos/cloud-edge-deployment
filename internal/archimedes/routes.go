package archimedes

import (
	"fmt"
	"net/http"

	"github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

// Route names
const (
	registerServiceName         = "REGISTER_SERVICE"
	deleteServiceName           = "DELETE_SERVICE"
	registerServiceInstanceName = "REGISTER_SERVICE_INSTANCE"
	deleteServiceInstanceName   = "DELETE_SERVICE_INSTANCE"
	getAllServicesName          = "GET_ALL_SERVICES"
	getAllServiceInstancesName  = "GET_ALL_SERVICE_INSTANCES"
	getServiceInstanceName      = "GET_SERVICE_INSTANCE"
	getInstanceName             = "GET_INSTANCE"
	discoverName                = "DISCOVER"
	whoAreYouName               = "WHO_ARE_YOU"
	getTableName                = "GET_TABLE"
	resolveName                 = "RESOLVE"
	redirectName                = "REDIRECT"
	removeRedirectName          = "REMOVE_REDIRECT"
	getRedirectedName           = "GET_REDIRECTED"
	resolveLocallyName          = "RESOLVE_LOCALLY"
	setResolvingAnswerName      = "SET_RESOLVE_ANSWER"
)

// Path variables
const (
	serviceIdPathVar  = "serviceId"
	instanceIdPathVar = "instanceId"
)

var (
	_serviceIdPathVarFormatted  = fmt.Sprintf(utils.PathVarFormat, serviceIdPathVar)
	_instanceIdPathVarFormatted = fmt.Sprintf(utils.PathVarFormat, instanceIdPathVar)

	servicesRoute        = archimedes.ServicesPath
	serviceRoute         = fmt.Sprintf(archimedes.ServicePath, _serviceIdPathVarFormatted)
	serviceInstanceRoute = fmt.Sprintf(archimedes.ServiceInstancePath, _serviceIdPathVarFormatted,
		_instanceIdPathVarFormatted)
	instanceRoute           = fmt.Sprintf(archimedes.InstancePath, _instanceIdPathVarFormatted)
	discoverRoute           = archimedes.DiscoverPath
	whoAreYouRoute          = archimedes.WhoAreYouPath
	tableRoute              = archimedes.TablePath
	resolveRoute            = archimedes.ResolvePath
	resolveLocallyRoute     = archimedes.ResolveLocallyPath
	redirectRoute           = fmt.Sprintf(archimedes.RedirectPath, _serviceIdPathVarFormatted)
	redirectedRoute         = fmt.Sprintf(archimedes.RedirectedPath, _serviceIdPathVarFormatted)
	setResolvingAnswerRoute = fmt.Sprintf(archimedes.SetResolvingAnswerPath)
)

var Routes = []utils.Route{
	{
		Name:        setResolvingAnswerName,
		Method:      http.MethodPost,
		Pattern:     setResolvingAnswerRoute,
		HandlerFunc: setResolutionAnswerHandler,
	},

	{
		Name:        resolveLocallyName,
		Method:      http.MethodPost,
		Pattern:     resolveLocallyRoute,
		HandlerFunc: resolveLocallyHandler,
	},

	{
		Name:        getRedirectedName,
		Method:      http.MethodGet,
		Pattern:     redirectedRoute,
		HandlerFunc: getRedirectedHandler,
	},

	{
		Name:        redirectName,
		Method:      http.MethodPost,
		Pattern:     redirectRoute,
		HandlerFunc: redirectHandler,
	},

	{
		Name:        removeRedirectName,
		Method:      http.MethodDelete,
		Pattern:     redirectRoute,
		HandlerFunc: removeRedirectionHandler,
	},

	{
		Name:        registerServiceName,
		Method:      http.MethodPost,
		Pattern:     serviceRoute,
		HandlerFunc: registerServiceHandler,
	},

	{
		Name:        deleteServiceName,
		Method:      http.MethodDelete,
		Pattern:     serviceRoute,
		HandlerFunc: deleteServiceHandler,
	},

	{
		Name:        registerServiceInstanceName,
		Method:      http.MethodPost,
		Pattern:     serviceInstanceRoute,
		HandlerFunc: registerServiceInstanceHandler,
	},

	{
		Name:        deleteServiceInstanceName,
		Method:      http.MethodDelete,
		Pattern:     serviceInstanceRoute,
		HandlerFunc: deleteServiceInstanceHandler,
	},

	{
		Name:        getAllServicesName,
		Method:      http.MethodGet,
		Pattern:     servicesRoute,
		HandlerFunc: getAllServicesHandler,
	},

	{
		Name:        getAllServiceInstancesName,
		Method:      http.MethodGet,
		Pattern:     serviceRoute,
		HandlerFunc: getAllServiceInstancesHandler,
	},

	{
		Name:        getInstanceName,
		Method:      http.MethodGet,
		Pattern:     instanceRoute,
		HandlerFunc: getInstanceHandler,
	},

	{
		Name:        getServiceInstanceName,
		Method:      http.MethodGet,
		Pattern:     serviceInstanceRoute,
		HandlerFunc: getServiceInstanceHandler,
	},

	{
		Name:        discoverName,
		Method:      http.MethodPost,
		Pattern:     discoverRoute,
		HandlerFunc: discoverHandler,
	},

	{
		Name:        whoAreYouName,
		Method:      http.MethodGet,
		Pattern:     whoAreYouRoute,
		HandlerFunc: whoAreYouHandler,
	},

	{
		Name:        getTableName,
		Method:      http.MethodGet,
		Pattern:     tableRoute,
		HandlerFunc: getServicesTableHandler,
	},

	{
		Name:        resolveName,
		Method:      http.MethodPost,
		Pattern:     resolveRoute,
		HandlerFunc: resolveHandler,
	},
}
