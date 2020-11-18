package deployer

import (
	"fmt"
	"net/http"

	"github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

// Route names
const (
	getDeploymentsName             = "GET_DEPLOYMENTS"
	registerDeploymentName         = "REGISTER_DEPLOYMENT"
	registerDeploymentInstanceName = "REGISTER_DEPLOYMENT_INSTANCE"
	deleteDeploymentName           = "DELETE_DEPLOYMENT"
	whoAreYouName                  = "WHO_ARE_YOU"
	addNodeName                    = "ADD_NODE"
	setAlternativesName            = "SET_ALTERNATIVES"
	deadChildName                  = "DEAD_CHILD"
	deleteDeploymentChildName      = "DELETE_DEPLOYMENT_CHILD"
	iAmYourParentName              = "I_AM_YOUR_PARENT"
	getHierarchyTableName          = "GET_TABLE"
	parentAliveName                = "PARENT_ALIVE"
	extendDeploymentToName         = "EXTEND_DEPLOYMENT_TO"
	setGrandparentName             = "SET_GRANDPARENT"
	fallbackName                   = "FALLBACK"
	getFallbackIdName              = "GET_FALLBACK"
	hasDeploymentName              = "HAS_DEPLOYMENT"
	propagateLocationToHorizonName = "PROPAGATE_LOCATION_TO_HORIZON"

	// scheduler
	heartbeatDeploymentInstanceName         = "HEARTBEAT_DEPLOYMENT_INSTANCE"
	registerHeartbeatDeploymentInstanceName = "REGISTER_HEARTBEAT"
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
	_deployerIdPathVarFormatted   = fmt.Sprintf(utils.PathVarFormat, nodeIdPathVar)

	deploymentsRoute           = deployer.DeploymentsPath
	deploymentRoute            = fmt.Sprintf(deployer.DeploymentPath, _deploymentIdPathVarFormatted)
	addNodeRoute               = deployer.AddNodePath
	whoAreYouRoute             = deployer.WhoAreYouPath
	setAlternativesRoute       = fmt.Sprintf(deployer.SetAlternativesPath, _deployerIdPathVarFormatted)
	deadChildRoute             = fmt.Sprintf(deployer.DeadChildPath, _deploymentIdPathVarFormatted, _deployerIdPathVarFormatted)
	deploymentChildRoute       = fmt.Sprintf(deployer.DeploymentChildPath, _deploymentIdPathVarFormatted, _deployerIdPathVarFormatted)
	iAmYourParentRoute         = fmt.Sprintf(deployer.IAmYourParentPath, _deploymentIdPathVarFormatted)
	hierarchyTableRoute        = deployer.HierarchyTablePath
	extendDeploymentToRoute    = fmt.Sprintf(deployer.ExtendDeploymentToPath, _deploymentIdPathVarFormatted, _deployerIdPathVarFormatted)
	setGrandparentRoute             = fmt.Sprintf(deployer.SetGrandparentPath, _deploymentIdPathVarFormatted)
	fallbackRoute                   = fmt.Sprintf(deployer.FallbackPath, _deploymentIdPathVarFormatted)
	getFallbackRoute                = deployer.GetFallbackIdPath
	hasDeploymentRoute              = fmt.Sprintf(deployer.HasDeploymentPath, _deploymentIdPathVarFormatted)
	propagateLocationToHorzionRoute = fmt.Sprintf(deployer.PropagateLocationToHorizon, _deploymentIdPathVarFormatted)

	// scheduler
	deploymentInstanceAliveRoute = fmt.Sprintf(deployer.DeploymentInstanceAlivePath, _deploymentIdPathVarFormatted,
		_instanceIdPathVarFormatted)
	deploymentInstanceRoute = fmt.Sprintf(deployer.DeploymentInstancePath, _deploymentIdPathVarFormatted,
		_instanceIdPathVarFormatted)
	parentAliveRoute = fmt.Sprintf(deployer.ParentAlivePath, _deployerIdPathVarFormatted)
)

var Routes = []utils.Route{

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
		Name:        getFallbackIdName,
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
		Name:        addNodeName,
		Method:      http.MethodPost,
		Pattern:     addNodeRoute,
		HandlerFunc: addNodeHandler,
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
