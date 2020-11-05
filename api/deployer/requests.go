package deployer

import (
	"github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/golang/geo/s2"
)

const (
	ParentIdx      = 0
	GrandparentIdx = 1
)

type (
	RegisterDeploymentRequestBody struct {
		DeploymentConfig *DeploymentDTO
		Exploring        bool
	}
	AddNodeRequestBody   = string
	DeadChildRequestBody struct {
		Grandchild   *utils.Node
		Alternatives map[string]*utils.Node
		Locations    []s2.CellID
	}
	IAmYourParentRequestBody = []*utils.Node
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
		Parent    *utils.Node
		Children  []*utils.Node
		Locations []s2.CellID
		ToExclude map[string]interface{}
	}
	ExtendDeploymentRequestBody struct {
		Exploring bool
		Config    *ExtendDeploymentConfig
	}
	PropagateLocationToHorizonRequestBody struct {
		TTL      int8
		ChildId  string
		Location s2.CellID
	}
)
