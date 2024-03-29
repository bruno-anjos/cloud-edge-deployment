package deployer

import (
	"fmt"
	"net/http"

	"github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
)

// Route names
const (
	getDeploymentsName          = "GET_DEPLOYMENTS"
	registerDeploymentName      = "REGISTER_DEPLOYMENT"
	registerServiceInstanceName = "REGISTER_SERVICE_INSTANCE"
	deleteDeploymentName        = "DELETE_DEPLOYMENT"
	whoAreYouName               = "WHO_ARE_YOU"
	addNodeName                 = "ADD_NODE"
	setAlternativesName         = "SET_ALTERNATIVES"
	deadChildName               = "DEAD_CHILD"
	takeChildName               = "TAKE_CHILD"
	deleteDeploymentChildName   = "DELETE_DEPLOYMENT_CHILD"
	iAmYourParentName           = "I_AM_YOUR_PARENT"
	getHierarchyTableName       = "GET_TABLE"
	parentAliveName             = "PARENT_ALIVE"
	canTakeChildName            = "CAN_TAKE_CHILD"
	migrateDeploymentName       = "MIGRATE_DEPLOYMENT"
	extendDeploymentToName      = "EXTEND_DEPLOYMENT_TO"
	shortenDeploymentFromName   = "SHORTEN_DEPLOYMENT_FROM"
	setGrandparentName          = "SET_GRANDPARENT"
	canTakeParentName           = "CAN_TAKE_PARENT"
	fallbackName                = "FALLBACK"
	resolveUpTheTreeName        = "RESOLVE_UP_THE_TREE"
	startResolveUpTheTreeName   = "START_RESOLVE_UP_THE_TREE"
	redirectDownTheTreeName     = "REDIRECT_DOWN_THE_TREE"
	getFallbackIdName           = "GET_FALLBACK"
	hasDeploymentName           = "HAS_DEPLOYMENT"
	terminalLocationName        = "TERMINAL_LOCATION"
	exploringName               = "EXPLORING"

	// scheduler
	heartbeatServiceInstanceName         = "HEARTBEAT_SERVICE_INSTANCE"
	registerHeartbeatServiceInstanceName = "REGISTER_HEARTBEAT"
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
	migrateDeploymentRoute     = fmt.Sprintf(deployer.MigrateDeploymentPath, _deploymentIdPathVarFormatted)
	extendDeploymentToRoute    = fmt.Sprintf(deployer.ExtendServiceToPath, _deploymentIdPathVarFormatted, _deployerIdPathVarFormatted)
	shortenDeploymentFromRoute = fmt.Sprintf(deployer.ShortenServiceFromPath, _deploymentIdPathVarFormatted,
		_deployerIdPathVarFormatted)
	canTakeChildRoute = fmt.Sprintf(deployer.CanTakeChildPath, _deploymentIdPathVarFormatted,
		_deployerIdPathVarFormatted)
	setGrandparentRoute        = fmt.Sprintf(deployer.SetGrandparentPath, _deploymentIdPathVarFormatted)
	canTakeParentRoute         = fmt.Sprintf(deployer.CanTakeParentPath, _deploymentIdPathVarFormatted, _deployerIdPathVarFormatted)
	fallbackRoute              = fmt.Sprintf(deployer.FallbackPath, _deploymentIdPathVarFormatted)
	resolveUpTheTreeRoute      = fmt.Sprintf(deployer.ResolveUpTheTreePath, _deploymentIdPathVarFormatted)
	startResolveUpTheTreeRoute = fmt.Sprintf(deployer.StartResolveUpTheTreePath, _deploymentIdPathVarFormatted)
	redirectDownTheTreeRoute   = fmt.Sprintf(deployer.RedirectDownTheTreePath, _deploymentIdPathVarFormatted)
	getFallbackRoute           = deployer.GetFallbackIdPath
	hasDeploymentRoute         = fmt.Sprintf(deployer.HasDeploymentPath, _deploymentIdPathVarFormatted)
	terminalLocationRoute      = fmt.Sprintf(deployer.TerminalLocationPath, _deploymentIdPathVarFormatted)
	setExploringRoute          = fmt.Sprintf(deployer.SetExploringPath, _deploymentIdPathVarFormatted, _deployerIdPathVarFormatted)

	// scheduler
	deploymentInstanceAliveRoute = fmt.Sprintf(deployer.DeploymentInstanceAlivePath, _deploymentIdPathVarFormatted,
		_instanceIdPathVarFormatted)
	deploymentInstanceRoute = fmt.Sprintf(deployer.DeploymentInstancePath, _deploymentIdPathVarFormatted,
		_instanceIdPathVarFormatted)
	parentAliveRoute = fmt.Sprintf(deployer.ParentAlivePath, _deployerIdPathVarFormatted)
)

var Routes = []utils.Route{

	{
		Name:        exploringName,
		Method:      http.MethodPost,
		Pattern:     setExploringRoute,
		HandlerFunc: setExploringHandler,
	},

	{
		Name:        terminalLocationName,
		Method:      http.MethodPost,
		Pattern:     terminalLocationRoute,
		HandlerFunc: terminalLocationHandler,
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
		Name:        redirectDownTheTreeName,
		Method:      http.MethodGet,
		Pattern:     redirectDownTheTreeRoute,
		HandlerFunc: redirectClientDownTheTreeHandler,
	},

	{
		Name:        startResolveUpTheTreeName,
		Method:      http.MethodPost,
		Pattern:     startResolveUpTheTreeRoute,
		HandlerFunc: startResolveUpTheTreeHandler,
	},

	{
		Name:        resolveUpTheTreeName,
		Method:      http.MethodPost,
		Pattern:     resolveUpTheTreeRoute,
		HandlerFunc: resolveUpTheTreeHandler,
	},

	{
		Name:        fallbackName,
		Method:      http.MethodPost,
		Pattern:     fallbackRoute,
		HandlerFunc: fallbackHandler,
	},

	{
		Name:        canTakeParentName,
		Method:      http.MethodGet,
		Pattern:     canTakeParentRoute,
		HandlerFunc: canTakeParentHandler,
	},

	{
		Name:        setGrandparentName,
		Method:      http.MethodPost,
		Pattern:     setGrandparentRoute,
		HandlerFunc: setGrandparentHandler,
	},

	{
		Name:        canTakeChildName,
		Method:      http.MethodGet,
		Pattern:     canTakeChildRoute,
		HandlerFunc: canTakeChildHandler,
	},

	{
		Name:        parentAliveName,
		Method:      http.MethodPost,
		Pattern:     parentAliveRoute,
		HandlerFunc: parentAliveHandler,
	},

	{
		Name:        shortenDeploymentFromName,
		Method:      http.MethodPost,
		Pattern:     shortenDeploymentFromRoute,
		HandlerFunc: shortenDeploymentFromHandler,
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
		Name:        migrateDeploymentName,
		Method:      http.MethodPost,
		Pattern:     migrateDeploymentRoute,
		HandlerFunc: migrateDeploymentHandler,
	},

	{
		Name:        takeChildName,
		Method:      http.MethodPost,
		Pattern:     deploymentChildRoute,
		HandlerFunc: takeChildHandler,
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
		Name:        heartbeatServiceInstanceName,
		Method:      http.MethodPut,
		Pattern:     deploymentInstanceAliveRoute,
		HandlerFunc: heartbeatServiceInstanceHandler,
	},

	{
		Name:        registerHeartbeatServiceInstanceName,
		Method:      http.MethodPost,
		Pattern:     deploymentInstanceAliveRoute,
		HandlerFunc: registerHeartbeatServiceInstanceHandler,
	},

	{
		Name:        registerServiceInstanceName,
		Method:      http.MethodPost,
		Pattern:     deploymentInstanceRoute,
		HandlerFunc: registerServiceInstanceHandler,
	},
}
