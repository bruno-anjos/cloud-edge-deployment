package deployment

import (
	"strconv"
	"sync"
	"time"

	autonomicAPI "github.com/bruno-anjos/cloud-edge-deployment/api/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/actions"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/scheduler"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	"github.com/golang/geo/s2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	blacklistDuration = 30 * time.Minute
)

type (
	nodeWithLocation struct {
		Node     *utils.Node
		Location s2.CellID
	}

	exploringMapValue = int

	Deployment struct {
		DeploymentId string
		Strategy     strategy
		Children     *sync.Map
		Parent       *utils.Node
		Suspected    *sync.Map
		Environment  *environment.Environment
		Blacklist    *sync.Map
		Exploring    *sync.Map
		DepthFactor  float64
		autoFactory  autonomic.ClientFactory
		archFactory  archimedes.ClientFactory
		deplFactory  deployer.ClientFactory
		schedFactory scheduler.ClientFactory
	}
)

var (
	Myself *utils.Node
)

func init() {
	Myself = utils.NodeFromEnv()
}

func New(deploymentId, strategyId string, suspected *sync.Map, depthFactor float64,
	env *environment.Environment, autoFactory autonomic.ClientFactory) (*Deployment, error) {
	s := &Deployment{
		Children:     &sync.Map{},
		Parent:       nil,
		Suspected:    suspected,
		Environment:  env,
		DeploymentId: deploymentId,
		Blacklist:    &sync.Map{},
		Exploring:    &sync.Map{},
		DepthFactor:  depthFactor,
		autoFactory:  autoFactory,
	}

	var strat strategy
	switch strategyId {
	case autonomic.StrategyLoadBalanceId:
		strat = newDefaultLoadBalanceStrategy(s)
	case autonomic.StrategyIdealLatencyId:
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

func (a *Deployment) AddChild(child *utils.Node, location s2.CellID) {
	node := &nodeWithLocation{
		Node:     child,
		Location: location,
	}
	a.Children.Store(child.Id, node)
}

func (a *Deployment) RemoveChild(childId string) {
	a.Children.Delete(childId)

	_, ok := a.Exploring.Load(childId)
	if ok {
		a.BlacklistNodes(Myself.Id, childId)
	}
}

func (a *Deployment) AddSuspectedChild(childId string) {
	a.Suspected.Store(childId, nil)
}

func (a *Deployment) removeSuspectedChild(childId string) {
	a.Suspected.Delete(childId)
}

func (a *Deployment) SetParent(parent *utils.Node) {
	a.Parent = parent
}

func (a *Deployment) GenerateAction() actions.Action {
	return a.Strategy.Optimize()
}

func (a *Deployment) ToDTO() *autonomicAPI.DeploymentDTO {
	var children []string
	a.Children.Range(func(key, value interface{}) bool {
		childId := key.(string)
		children = append(children, childId)
		return true
	})

	return &autonomicAPI.DeploymentDTO{
		DeploymentId: a.DeploymentId,
		StrategyId:   a.Strategy.GetId(),
		Children:     children,
		Parent:       a.Parent,
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

func (a *Deployment) BlacklistNodes(origin string, nodes ...string) {
	log.Debugf("blacklisting %+v", nodes)
	for _, node := range nodes {
		a.Blacklist.Store(node, nil)
	}

	autoClient := a.autoFactory.New(a.Parent.Addr + ":" + strconv.Itoa(autonomic.Port))
	if a.Parent != nil && origin != a.Parent.Id {
		autoClient.BlacklistNodes(a.DeploymentId, Myself.Id, nodes...)
	}
	a.Children.Range(func(key, value interface{}) bool {
		childId := key.(string)
		if childId == origin {
			return true
		}
		log.Debugf("telling %s to blacklist %+v for %s", childId, nodes, a.DeploymentId)
		nodeWithLoc := value.(nodeWithLocation)
		autoClient.SetHostPort(nodeWithLoc.Node.Addr + ":" + strconv.Itoa(autonomic.Port))
		autoClient.BlacklistNodes(a.DeploymentId, Myself.Id, nodes...)
		return true
	})

	go func() {
		blacklistTimer := time.NewTimer(blacklistDuration)
		<-blacklistTimer.C
		log.Debugf("removing %+v from blacklist", nodes)
		for _, node := range nodes {
			a.Blacklist.Delete(node)
		}
	}()
}

func (a *Deployment) removeFromBlacklist(nodeId string) {
	a.Blacklist.Delete(nodeId)
}

func (a *Deployment) SetExploreSuccess(childId string) bool {
	a.Exploring.Delete(childId)
	log.Debugf("explored %s successfully", childId)
	return true
}

func (a *Deployment) setNodeAsExploring(nodeId string) {
	log.Debugf("exploring child %s", nodeId)
	a.Exploring.Store(nodeId, nil)
}
