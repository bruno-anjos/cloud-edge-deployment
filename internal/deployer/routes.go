package deployer

import (
	"fmt"
	"net/http"

	"github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
)

// Route names
const (
	getDeploymentsName             = "GET_DEPLOYMENTS"
	registerDeploymentName         = "REGISTER_DEPLOYMENT"
	registerDeploymentInstanceName = "REGISTER_DEPLOYMENT_INSTANCE"
	deleteDeploymentName           = "DELETE_DEPLOYMENT"
	whoAreYouName                  = "WHO_ARE_YOU"
	setAlternativesName            = "SET_ALTERNATIVES"
	deadChildName                  = "DEAD_CHILD"
	deleteDeploymentChildName      = "DELETE_DEPLOYMENT_CHILD"
	iAmYourParentName              = "I_AM_YOUR_PARENT"
	getHierarchyTableName          = "GET_TABLE"
	parentAliveName                = "PARENT_ALIVE"
	extendDeploymentToName         = "EXTEND_DEPLOYMENT_TO"
	setGrandparentName             = "SET_GRANDPARENT"
	fallbackName                   = "FALLBACK"
	getFallbackIDName              = "GET_FALLBACK"
	hasDeploymentName              = "HAS_DEPLOYMENT"
	propagateLocationToHorizonName = "PROPAGATE_LOCATION_TO_HORIZON"

	// scheduler
	heartbeatDeploymentInstanceName         = "HEARTBEAT_DEPLOYMENT_INSTANCE"
	registerHeartbeatDeploymentInstanceName = "REGISTER_HEARTBEAT"
)

// Path variables
const (
	deploymentIDPathVar = "deploymentId"
	nodeIDPathVar       = "nodeId"
	instanceIDPathVar   = "instanceId"
)

var (
	_deploymentIDPathVarFormatted = fmt.Sprintf(servers.PathVarFormat, deploymentIDPathVar)
	_instanceIDPathVarFormatted   = fmt.Sprintf(servers.PathVarFormat, instanceIDPathVar)
	_deployerIDPathVarFormatted   = fmt.Sprintf(servers.PathVarFormat, nodeIDPathVar)

	deploymentsRoute     = deployer.DeploymentsPath
	deploymentRoute      = fmt.Sprintf(deployer.DeploymentPath, _deploymentIDPathVarFormatted)
	whoAreYouRoute       = deployer.WhoAreYouPath
	setAlternativesRoute = fmt.Sprintf(deployer.SetAlternativesPath, _deployerIDPathVarFormatted)
	deadChildRoute       = fmt.Sprintf(deployer.DeadChildPath, _deploymentIDPathVarFormatted,
		_deployerIDPathVarFormatted)
	deploymentChildRoute = fmt.Sprintf(deployer.DeploymentChildPath, _deploymentIDPathVarFormatted,
		_deployerIDPathVarFormatted)
	iAmYourParentRoute              = fmt.Sprintf(deployer.IAmYourParentPath, _deploymentIDPathVarFormatted)
	hierarchyTableRoute             = deployer.HierarchyTablePath
	extendDeploymentToRoute         = fmt.Sprintf(deployer.ExtendDeploymentToPath, _deploymentIDPathVarFormatted)
	setGrandparentRoute             = fmt.Sprintf(deployer.SetGrandparentPath, _deploymentIDPathVarFormatted)
	fallbackRoute                   = fmt.Sprintf(deployer.FallbackPath, _deploymentIDPathVarFormatted)
	getFallbackRoute                = deployer.GetFallbackIDPath
	hasDeploymentRoute              = fmt.Sprintf(deployer.HasDeploymentPath, _deploymentIDPathVarFormatted)
	propagateLocationToHorzionRoute = fmt.Sprintf(deployer.PropagateLocationToHorizon, _deploymentIDPathVarFormatted)

	// scheduler
	deploymentInstanceAliveRoute = fmt.Sprintf(deployer.DeploymentInstanceAlivePath, _deploymentIDPathVarFormatted,
		_instanceIDPathVarFormatted)
	deploymentInstanceRoute = fmt.Sprintf(deployer.DeploymentInstancePath, _deploymentIDPathVarFormatted,
		_instanceIDPathVarFormatted)
	parentAliveRoute = fmt.Sprintf(deployer.ParentAlivePath, _deployerIDPathVarFormatted)
)

var Routes = []servers.Route{

	{
		Name:        propagateLocationToHorizonName,
		Method:      http.MethodPost,
		Pattern:     propagateLocationToHorzionRoute,
		HandlerFunc: propagateLocationToHorizonHandler,
	},

	{
		Name:        hasDeploymentName,
		Method:      http.MethodGet,
		Pattern:     hasDeploymentRoute,
		HandlerFunc: hasDeploymentHandler,
	},

	{
		Name:        getFallbackIDName,
		Method:      http.MethodGet,
		Pattern:     getFallbackRoute,
		HandlerFunc: getFallbackHandler,
	},

	{
		Name:        fallbackName,
		Method:      http.MethodPost,
		Pattern:     fallbackRoute,
		HandlerFunc: fallbackHandler,
	},

	{
		Name:        setGrandparentName,
		Method:      http.MethodPost,
		Pattern:     setGrandparentRoute,
		HandlerFunc: setGrandparentHandler,
	},

	{
		Name:        parentAliveName,
		Method:      http.MethodPost,
		Pattern:     parentAliveRoute,
		HandlerFunc: parentAliveHandler,
	},

	{
		Name:        extendDeploymentToName,
		Method:      http.MethodPost,
		Pattern:     extendDeploymentToRoute,
		HandlerFunc: extendDeploymentToHandler,
	},

	{
		Name:        deadChildName,
		Method:      http.MethodPost,
		Pattern:     deadChildRoute,
		HandlerFunc: deadChildHandler,
	},

	{
		Name:        deleteDeploymentChildName,
		Method:      http.MethodDelete,
		Pattern:     deploymentChildRoute,
		HandlerFunc: childDeletedDeploymentHandler,
	},

	{
		Name:        iAmYourParentName,
		Method:      http.MethodPost,
		Pattern:     iAmYourParentRoute,
		HandlerFunc: iAmYourParentHandler,
	},

	{
		Name:        getHierarchyTableName,
		Method:      http.MethodGet,
		Pattern:     hierarchyTableRoute,
		HandlerFunc: getHierarchyTableHandler,
	},

	{
		Name:        getDeploymentsName,
		Method:      http.MethodGet,
		Pattern:     deploymentsRoute,
		HandlerFunc: getDeploymentsHandler,
	},

	{
		Name:        registerDeploymentName,
		Method:      http.MethodPost,
		Pattern:     deploymentsRoute,
		HandlerFunc: registerDeploymentHandler,
	},

	{
		Name:        deleteDeploymentName,
		Method:      http.MethodDelete,
		Pattern:     deploymentRoute,
		HandlerFunc: deleteDeploymentHandler,
	},

	{
		Name:        whoAreYouName,
		Method:      http.MethodGet,
		Pattern:     whoAreYouRoute,
		HandlerFunc: whoAreYouHandler,
	},

	{
		Name:        setAlternativesName,
		Method:      http.MethodPost,
		Pattern:     setAlternativesRoute,
		HandlerFunc: setAlternativesHandler,
	},

	{
		Name:        heartbeatDeploymentInstanceName,
		Method:      http.MethodPut,
		Pattern:     deploymentInstanceAliveRoute,
		HandlerFunc: heartbeatDeploymentInstanceHandler,
	},

	{
		Name:        registerHeartbeatDeploymentInstanceName,
		Method:      http.MethodPost,
		Pattern:     deploymentInstanceAliveRoute,
		HandlerFunc: registerHeartbeatDeploymentInstanceHandler,
	},

	{
		Name:        registerDeploymentInstanceName,
		Method:      http.MethodPost,
		Pattern:     deploymentInstanceRoute,
		HandlerFunc: registerDeploymentInstanceHandler,
	},
}
