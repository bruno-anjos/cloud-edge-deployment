package autonomic

import (
	"fmt"
	"net/http"

	"github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

// Route names
const (
	registerDeploymentName    = "REGISTER_DEPLOYMENT"
	deleteDeploymentName      = "DELETE_DEPLOYMENT"
	getAllDeploymentsName     = "GET_ALL_DEPLOYMENTS"
	addDeploymentChildName    = "ADD_DEPLOYMENT_CHILD"
	removeDeploymentChildName = "REMOVE_DEPLOYMENT_CHILD"
	setDeploymentParentName   = "SET_DEPLOYMENT_PARENT"
	isNodeInVicinityName      = "IS_NODE_IN_VICINITY"
	closestNodeName           = "CLOSEST_NODE"
	getVicinityName           = "GET_VICINITY"
	getMyLocationName         = "GET_MY_LOCATION"
	getLoadName               = "GET_LOAD"
	exploredSuccessfullyName  = "EXPLORED_SUCCESSFULLY"
	blacklistName             = "BLACKLIST"
)

// Path variables
const (
	deploymentIdPathVar = "deploymentId"
	childIdPathVar      = "childId"
	nodeIdPathVar       = "nodeId"
)

var (
	_deploymentIdPathVarFormatted = fmt.Sprintf(utils.PathVarFormat, deploymentIdPathVar)
	_childIdPathVarFormatted      = fmt.Sprintf(utils.PathVarFormat, childIdPathVar)
	_nodeIdPathVarFormatted       = fmt.Sprintf(utils.PathVarFormat, nodeIdPathVar)

	deploymentsRoute              = autonomic.DeploymentsPath
	deploymentRoute               = fmt.Sprintf(autonomic.DeploymentPath, _deploymentIdPathVarFormatted)
	deploymentChildRoute          = fmt.Sprintf(autonomic.DeploymentChildPath, _deploymentIdPathVarFormatted)
	deploymentChildWithChildRoute = fmt.Sprintf(autonomic.DeploymentChildPath, _deploymentIdPathVarFormatted, _childIdPathVarFormatted)
	deploymentParentRoute         = fmt.Sprintf(autonomic.DeploymentParentPath, _deploymentIdPathVarFormatted)
	isNodeInVicinityRoute         = fmt.Sprintf(autonomic.IsNodeInVicinityPath, _nodeIdPathVarFormatted)
	closestNodeRoute              = autonomic.ClosestNodePath
	getVicinityRoute              = autonomic.VicinityPath
	getMyLocationRoute            = autonomic.MyLocationPath
	getLoadRoute                  = fmt.Sprintf(autonomic.LoadPath, _deploymentIdPathVarFormatted)
	exploredSuccessfullyRoute     = fmt.Sprintf(autonomic.ExplorePath, _deploymentIdPathVarFormatted, _childIdPathVarFormatted)
	blacklistRoute                = fmt.Sprintf(autonomic.BlacklistPath, _deploymentIdPathVarFormatted)
)

var Routes = []utils.Route{
	{
		Name:        blacklistName,
		Method:      http.MethodPost,
		Pattern:     blacklistRoute,
		HandlerFunc: blacklistNodeHandler,
	},

	{
		Name:        exploredSuccessfullyName,
		Method:      http.MethodPost,
		Pattern:     exploredSuccessfullyRoute,
		HandlerFunc: setExploreSuccessfullyHandler,
	},

	{
		Name:        getLoadName,
		Method:      http.MethodGet,
		Pattern:     getLoadRoute,
		HandlerFunc: getLoadForDeploymentHandler,
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
		Name:        setDeploymentParentName,
		Method:      http.MethodPost,
		Pattern:     deploymentParentRoute,
		HandlerFunc: setDeploymentParentHandler,
	},

	{
		Name:        addDeploymentChildName,
		Method:      http.MethodPost,
		Pattern:     deploymentChildRoute,
		HandlerFunc: addDeploymentChildHandler,
	},

	{
		Name:        removeDeploymentChildName,
		Method:      http.MethodDelete,
		Pattern:     deploymentChildWithChildRoute,
		HandlerFunc: removeDeploymentChildHandler,
	},

	{
		Name:        registerDeploymentName,
		Method:      http.MethodPost,
		Pattern:     deploymentRoute,
		HandlerFunc: addDeploymentHandler,
	},

	{
		Name:        deleteDeploymentName,
		Method:      http.MethodDelete,
		Pattern:     deploymentRoute,
		HandlerFunc: removeDeploymentHandler,
	},

	{
		Name:        getAllDeploymentsName,
		Method:      http.MethodGet,
		Pattern:     deploymentsRoute,
		HandlerFunc: getAllDeploymentsHandler,
	},
}
