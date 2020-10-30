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
	RegisterDeploymentRequestBody = DeploymentDTO
	AddNodeRequestBody            = string
	DeadChildRequestBody          = struct {
		Grandchild   *utils.Node
		Alternatives map[string]*utils.Node
		Location     s2.CellID
	}
	IAmYourParentRequestBody = []*utils.Node
	IAmYourChildRequestBody  = struct {
		Child *utils.Node
	}
	RegisterDeploymentInstanceRequestBody = archimedes.InstanceDTO
	AlternativesRequestBody               = []*utils.Node
	SetGrandparentRequestBody             = utils.Node
	FallbackRequestBody                   = struct {
		OrphanId       string
		OrphanLocation s2.CellID
	}
	StartResolveUpTheTreeRequestBody = archimedes.ToResolveDTO
	ResolveUpTheTreeRequestBody      = struct {
		Origin    string
		ToResolve *archimedes.ToResolveDTO
	}
	RedirectClientDownTheTreeRequestBody = s2.CellID
	ExtendDeploymentRequestBody          = struct {
		Parent    *utils.Node
		Children  []*utils.Node
		Exploring bool
		Location  s2.CellID
	}
	PropagateLocationToHorizonRequestBody = struct {
		TTL      int8
		ChildId  string
		Location s2.CellID
	}
)
