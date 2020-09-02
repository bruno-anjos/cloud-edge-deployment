package node_goals
/*
import (
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic"
	log "github.com/sirupsen/logrus"
)

const (
	loadDiffThreshold = 0.25
)

type LoadBalance struct {
	environment *autonomic.Environment
}

func NewLoadBalance(env *autonomic.Environment) *LoadBalance {
	return &LoadBalance{
		environment: env,
	}
}

func (l *LoadBalance) Optimize(optDomain Domain) (isAlreadyMax bool, optRange Range) {
	isAlreadyMax = true
	optRange = nil

	value, ok := l.environment.GetMetric(autonomic.METRIC_LOAD)
	if !ok {
		log.Debugf("no value for metric %s", autonomic.METRIC_LOAD)
		return
	}

	myLoad := value.(float64)

	value, ok = l.environment.GetMetric(autonomic.METRIC_LOAD_IN_VICINITY)
	if !ok {
		log.Debugf("no value for metric %s", autonomic.METRIC_LOAD_IN_VICINITY)
		return
	}

	loadsInVicinity := value.(map[string]float64)
	candidateLoads := map[string]float64{}
	var migrateCandidates []string
	for nodeId, load := range loadsInVicinity {
		diff := myLoad - load
		if diff >= loadDiffThreshold {
			migrateCandidates = append(migrateCandidates, nodeId)
			candidateLoads[nodeId] = diff
		}
	}

	l.balanceLoads(myLoad, candidateLoads)

	isAlreadyMax = false
	optRange = filterCandidates(optDomain, migrateCandidates)

	return
}

func (l *LoadBalance) GenerateAction(target string) autonomic.Action {
	return autonomic.NewAddServiceAction(target)
}

func (l *LoadBalance) balanceLoads(myLoad float64, candidateLoads map[string]float64) []autonomic.Action {
	panic("implement me")
	return nil
}
*/