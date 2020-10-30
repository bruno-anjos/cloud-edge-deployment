package deployment

import (
	"os"
	"sync"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	public "github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/golang/geo/s2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	blacklistDuration = 5 * time.Minute
)

type nodeWithLocation struct {
	NodeId   string
	Location s2.CellID
}

type Deployment struct {
	DeploymentId string
	Strategy     strategy
	Children     *sync.Map
	ParentId     string
	Suspected    *sync.Map
	Environment  *environment.Environment
	Blacklist    *sync.Map
}

var (
	myself *utils.Node
)

func init() {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	myself = &utils.Node{
		Id:   hostname,
		Addr: hostname,
	}
}

func New(deploymentId, strategyId string, suspected *sync.Map,
	env *environment.Environment) (*Deployment, error) {
	s := &Deployment{
		Children:     &sync.Map{},
		ParentId:     "",
		Suspected:    suspected,
		Environment:  env,
		DeploymentId: deploymentId,
		Blacklist:    &sync.Map{},
	}

	var strat strategy
	switch strategyId {
	case public.StrategyLoadBalanceId:
		strat = newDefaultLoadBalanceStrategy(s)
	case public.StrategyIdealLatencyId:
		strat = newDefaultIdealLatencyStrategy(s)
	default:
		return nil, errors.Errorf("invalid strategy: %s", strategyId)
	}

	dependencies := strat.GetDependencies()
	if dependencies != nil {
		for _, deploymentMetric := range dependencies {
			env.TrackMetric(deploymentMetric)
		}
	}

	s.Strategy = strat

	return s, nil
}

func (a *Deployment) AddChild(childId string, location s2.CellID) {
	node := &nodeWithLocation{
		NodeId:   childId,
		Location: location,
	}
	a.Children.Store(childId, node)
}

func (a *Deployment) RemoveChild(childId string) {
	a.Children.Delete(childId)
}

func (a *Deployment) AddSuspectedChild(childId string) {
	a.Suspected.Store(childId, nil)
}

func (a *Deployment) removeSuspectedChild(childId string) {
	a.Suspected.Delete(childId)
}

func (a *Deployment) SetParent(parentId string) {
	a.ParentId = parentId
}

func (a *Deployment) GenerateAction() actions.Action {
	return a.Strategy.Optimize()
}

func (a *Deployment) ToDTO() *autonomic.DeploymentDTO {
	var children []string
	a.Children.Range(func(key, value interface{}) bool {
		childId := key.(string)
		children = append(children, childId)
		return true
	})

	return &autonomic.DeploymentDTO{
		DeploymentId: a.DeploymentId,
		StrategyId:   a.Strategy.GetId(),
		Children:     children,
		ParentId:     a.ParentId,
	}
}

func (a *Deployment) GetLoad() float64 {
	metric := metrics.GetLoadPerDeployment(a.DeploymentId)
	value, ok := a.Environment.GetMetric(metric)
	if !ok {
		log.Debugf("no value for metric %s", metric)
		return 0
	}

	return value.(float64)
}

func (a *Deployment) BlacklistNode(nodeId string) {
	log.Debugf("blacklisting %s", nodeId)
	a.Blacklist.Store(nodeId, nil)
	go func() {
		blacklistTimer := time.NewTimer(blacklistDuration)
		<-blacklistTimer.C
		log.Debugf("removing %s from blacklist", nodeId)
		a.Blacklist.Delete(nodeId)
	}()
}

func (a *Deployment) removeFromBlacklist(nodeId string) {
	a.Blacklist.Delete(nodeId)
}
