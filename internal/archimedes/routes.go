package archimedes

import (
	"fmt"
	"net/http"

	"github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

// Route names
const (
	registerDeploymentName         = "REGISTER_DEPLOYMENT"
	deleteDeploymentName           = "DELETE_DEPLOYMENT"
	registerDeploymentInstanceName = "REGISTER_DEPLOYMENT_INSTANCE"
	deleteDeploymentInstanceName   = "DELETE_DEPLOYMENT_INSTANCE"
	getAllDeploymentsName          = "GET_ALL_DEPLOYMENTS"
	getAllDeploymentInstancesName  = "GET_ALL_DEPLOYMENT_INSTANCES"
	getDeploymentInstanceName      = "GET_DEPLOYMENT_INSTANCE"
	getInstanceName                = "GET_INSTANCE"
	discoverName                   = "DISCOVER"
	whoAreYouName                  = "WHO_ARE_YOU"
	getTableName                   = "GET_TABLE"
	resolveName                    = "RESOLVE"
	redirectName                   = "REDIRECT"
	removeRedirectName             = "REMOVE_REDIRECT"
	getRedirectedName              = "GET_REDIRECTED"
	resolveLocallyName             = "RESOLVE_LOCALLY"
	getLoadName                    = "GET_LOAD"
	getAvgClientLocationName       = "GET_AVG_CLIENT_LOCATION"
	setExploringLocationName       = "SET_EXPLORING_CLIENT_LOCATION"
	addDeploymentNodeName          = "ADD_DEPLOYMENT_NODE"
	removeDeploymentNodeName       = "REMOVE_DEPLOYMENT_NODE"
	willRedirectToYouName          = "WILL_REDIRECT_TO_YOU"
	stopRedirectingToYouName       = "STOP_REDIRECTING_TO_YOU"
	canRedirectToYouName           = "CAN_REDIRECT_TO_YOU"
)

// Path variables
const (
	deploymentIdPathVar = "deploymentId"
	nodeIdPathVar       = "nodeId"
	instanceIdPathVar   = "instanceId"
)

var (
	_deploymentIdPathVarFormatted = fmt.Sprintf(utils.PathVarFormat, deploymentIdPathVar)
	_instanceIdPathVarFormatted   = fmt.Sprintf(utils.PathVarFormat, instanceIdPathVar)
	_nodeIdPathVarFormatted       = fmt.Sprintf(utils.PathVarFormat, nodeIdPathVar)

	deploymentsRoute        = archimedes.DeploymentsPath
	deploymentRoute         = fmt.Sprintf(archimedes.DeploymentPath, _deploymentIdPathVarFormatted)
	deploymentInstanceRoute = fmt.Sprintf(archimedes.DeploymentInstancePath, _deploymentIdPathVarFormatted,
		_instanceIdPathVarFormatted)
	instanceRoute                   = fmt.Sprintf(archimedes.InstancePath, _instanceIdPathVarFormatted)
	discoverRoute                   = archimedes.DiscoverPath
	whoAreYouRoute                  = archimedes.WhoAreYouPath
	tableRoute                      = archimedes.TablePath
	resolveRoute                    = archimedes.ResolvePath
	resolveLocallyRoute             = archimedes.ResolveLocallyPath
	redirectRoute                   = fmt.Sprintf(archimedes.RedirectPath, _deploymentIdPathVarFormatted)
	redirectedRoute                 = fmt.Sprintf(archimedes.RedirectedPath, _deploymentIdPathVarFormatted)
	getLoadRoute                    = fmt.Sprintf(archimedes.LoadPath, _deploymentIdPathVarFormatted)
	getAvgClientLocationRoute       = fmt.Sprintf(archimedes.AvgClientLocationPath, _deploymentIdPathVarFormatted)
	setExploringClientLocationRoute = fmt.Sprintf(archimedes.ExploringClientLocationPath, _deploymentIdPathVarFormatted)
	addDeploymentNodeRoute          = fmt.Sprintf(archimedes.AddDeploymentNodePath, _deploymentIdPathVarFormatted)
	removeDeploymentNodeRoute       = fmt.Sprintf(archimedes.RemoveDeploymentNodePath, _deploymentIdPathVarFormatted,
		_nodeIdPathVarFormatted)
	redirectingToYou = fmt.Sprintf(archimedes.RedirectingToYouPath, _deploymentIdPathVarFormatted,
		_nodeIdPathVarFormatted)
)

var Routes = []utils.Route{
	{
		Name:        canRedirectToYouName,
		Method:      http.MethodGet,
		Pattern:     redirectingToYou,
		HandlerFunc: canRedirectToYouHandler,
	},

	{
		Name:        stopRedirectingToYouName,
		Method:      http.MethodDelete,
		Pattern:     redirectingToYou,
		HandlerFunc: stoppedRedirectingToYouHandler,
	},

	{
		Name:        willRedirectToYouName,
		Method:      http.MethodPost,
		Pattern:     redirectingToYou,
		HandlerFunc: willRedirectToYouHandler,
	},

	{
		Name:        removeDeploymentNodeName,
		Method:      http.MethodDelete,
		Pattern:     removeDeploymentNodeRoute,
		HandlerFunc: removeDeploymentNodeHandler,
	},

	{
		Name:        addDeploymentNodeName,
		Method:      http.MethodPost,
		Pattern:     addDeploymentNodeRoute,
		HandlerFunc: addDeploymentNodeHandler,
	},

	{
		Name:        setExploringLocationName,
		Method:      http.MethodPost,
		Pattern:     setExploringClientLocationRoute,
		HandlerFunc: setExploringClientLocationHandler,
	},

	{
		Name:        getAvgClientLocationName,
		Method:      http.MethodGet,
		Pattern:     getAvgClientLocationRoute,
		HandlerFunc: getClientCentroidsHandler,
	},

	{
		Name:        getLoadName,
		Method:      http.MethodGet,
		Pattern:     getLoadRoute,
		HandlerFunc: getLoadHandler,
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
		HandlerFunc: redirectServiceHandler,
	},

	{
		Name:        removeRedirectName,
		Method:      http.MethodDelete,
		Pattern:     redirectRoute,
		HandlerFunc: removeRedirectionHandler,
	},

	{
		Name:        registerDeploymentName,
		Method:      http.MethodPost,
		Pattern:     deploymentRoute,
		HandlerFunc: registerDeploymentHandler,
	},

	{
		Name:        deleteDeploymentName,
		Method:      http.MethodDelete,
		Pattern:     deploymentRoute,
		HandlerFunc: deleteDeploymentHandler,
	},

	{
		Name:        registerDeploymentInstanceName,
		Method:      http.MethodPost,
		Pattern:     deploymentInstanceRoute,
		HandlerFunc: registerDeploymentInstanceHandler,
	},

	{
		Name:        deleteDeploymentInstanceName,
		Method:      http.MethodDelete,
		Pattern:     deploymentInstanceRoute,
		HandlerFunc: deleteDeploymentInstanceHandler,
	},

	{
		Name:        getAllDeploymentsName,
		Method:      http.MethodGet,
		Pattern:     deploymentsRoute,
		HandlerFunc: getAllDeploymentsHandler,
	},

	{
		Name:        getAllDeploymentInstancesName,
		Method:      http.MethodGet,
		Pattern:     deploymentRoute,
		HandlerFunc: getAllDeploymentInstancesHandler,
	},

	{
		Name:        getInstanceName,
		Method:      http.MethodGet,
		Pattern:     instanceRoute,
		HandlerFunc: getInstanceHandler,
	},

	{
		Name:        getDeploymentInstanceName,
		Method:      http.MethodGet,
		Pattern:     deploymentInstanceRoute,
		HandlerFunc: getDeploymentInstanceHandler,
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
		HandlerFunc: getDeploymentsTableHandler,
	},

	{
		Name:        resolveName,
		Method:      http.MethodPost,
		Pattern:     resolveRoute,
		HandlerFunc: resolveHandler,
	},
}
