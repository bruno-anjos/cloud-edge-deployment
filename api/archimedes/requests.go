package archimedes

import (
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
)

type (
	RegisterDeploymentRequestBody struct {
		Deployment *DeploymentDTO
		Host       *utils.Node
	}
	RegisterDeploymentInstanceRequestBody = InstanceDTO
	ResolveRequestBody                    struct {
		ToResolve    *ToResolveDTO
		DeploymentId string
		Location     s2.CellID
		Id           string
		Redirects    []string
	}
	RedirectRequestBody            = redirectDTO
	SetResolutionAnswerRequestBody struct {
		Resolved *ResolvedDTO
		Id       string
	}
	SetExploringClientLocationRequestBody = []s2.CellID
	AddDeploymentNodeRequestBody          struct {
		Node      *utils.Node
		Location  s2.CellID
		Exploring bool
	}
)
