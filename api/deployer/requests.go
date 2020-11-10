package deployer

import (
	"github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/golang/geo/s2"
)

const (
	NotExploringTTL = -1
)

type (
	RegisterDeploymentRequestBody struct {
		DeploymentConfig *DeploymentDTO
		ExploringTTL     int
	}
	AddNodeRequestBody   = string
	DeadChildRequestBody struct {
		Grandchild   *utils.Node
		Alternatives map[string]*utils.Node
		Locations    []s2.CellID
	}
	IAmYourParentRequestBody struct {
		Parent *utils.Node
		Grandparent *utils.Node
	}
	IAmYourChildRequestBody  struct {
		Child *utils.Node
	}
	RegisterDeploymentInstanceRequestBody = archimedes.InstanceDTO
	AlternativesRequestBody               = []*utils.Node
	SetGrandparentRequestBody             = utils.Node
	FallbackRequestBody                   struct {
		OrphanId       string
		OrphanLocation s2.CellID
	}
	StartResolveUpTheTreeRequestBody = archimedes.ToResolveDTO
	ResolveUpTheTreeRequestBody      struct {
		Origin    string
		ToResolve *archimedes.ToResolveDTO
	}
	RedirectClientDownTheTreeRequestBody = s2.CellID
	ExtendDeploymentConfig               struct {
		Children  []*utils.Node
		Locations []s2.CellID
		ToExclude map[string]interface{}
	}
	ExtendDeploymentRequestBody struct {
		ExploringTTL int
		Config       *ExtendDeploymentConfig
	}
	PropagateLocationToHorizonRequestBody struct {
		TTL      int8
		ChildId  string
		Location s2.CellID
	}
)
