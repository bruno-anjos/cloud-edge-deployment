package deployer

import (
	"github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	publicUtils "github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
)

const (
	ParentIdx      = 0
	GrandparentIdx = 1
)

type (
	RegisterServiceRequestBody = DeploymentDTO
	AddNodeRequestBody         = string
	DeadChildRequestBody       = struct {
		Grandchild   *utils.Node
		Alternatives map[string]*utils.Node
		Location     *publicUtils.Location
	}
	IAmYourParentRequestBody = []*utils.Node
	IAmYourChildRequestBody  = struct {
		Child *utils.Node
	}
	RegisterServiceInstanceRequestBody = archimedes.InstanceDTO
	AlternativesRequestBody            = []*utils.Node
	SetGrandparentRequestBody          = utils.Node
	FallbackRequestBody                = struct {
		OrphanId       string
		OrphanLocation *publicUtils.Location
	}
	StartResolveUpTheTreeRequestBody = archimedes.ToResolveDTO
	ResolveUpTheTreeRequestBody      = struct {
		Origin    string
		ToResolve *archimedes.ToResolveDTO
	}
	RedirectClientDownTheTreeRequestBody = *publicUtils.Location
	ExtendDeploymentRequestBody          = struct {
		Parent    *utils.Node
		Children  []*utils.Node
		Exploring bool
	}
	PropagateLocationToHorizonRequestBody = struct {
		TTL      int8
		ChildId  string
		Location *publicUtils.Location
	}
)
