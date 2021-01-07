package environment

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"

	metrics "github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/metrics"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

type Environment struct {
	trackedMetrics *sync.Map
	metrics        *sync.Map
}

const (
	metricsFolder        = "metrics/"
	metricsFileExtension = ".met"
	nodeIPsFilepath      = "/node_ips.json"
)

func NewEnvironment() *Environment {
	env := &Environment{
		trackedMetrics: &sync.Map{},
		metrics:        &sync.Map{},
	}

	env.loadSimFile()

	return env
}

func (e *Environment) loadSimFile() {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	data, err := ioutil.ReadFile(metricsFolder + hostname + metricsFileExtension)
	if err != nil {
		panic(err)
	}

	var metricsMap map[string]interface{}

	err = json.Unmarshal(data, &metricsMap)
	if err != nil {
		panic(err)
	}

	for metricID, metricValue := range metricsMap {
		log.Debugf("loaded metric %s with value %v", metricID, metricValue)
		e.TrackMetric(metricID)

		if metricID == metrics.MetricLocationInVicinity {
			metricValue = loadVicinity(metricValue)
		}

		e.setMetric(metricID, metricValue)
	}
}

func (e *Environment) TrackMetric(metricID string) {
	_, loaded := e.trackedMetrics.LoadOrStore(metricID, nil)
	if loaded {
		return
	}

	registerMetricInLowerAPI(metricID)
}

func (e *Environment) GetMetric(metricID string) (value interface{}, ok bool) {
	return e.metrics.Load(metricID)
}

func (e *Environment) setMetric(metricID string, value interface{}) {
	e.metrics.Store(metricID, value)
}

func (e *Environment) deleteMetric(metricID string) {
	e.metrics.Delete(metricID)
}

func (e *Environment) copy() (copy *Environment) {
	newMap := &sync.Map{}
	copy = &Environment{metrics: newMap}

	e.metrics.Range(func(key, value interface{}) bool {
		newMap.Store(key, value)

		return true
	})

	return
}

func loadVicinity(metricValue interface{}) interface{} {
	var locationsInVicinity metrics.VicinityMetric

	err := mapstructure.Decode(metricValue, &locationsInVicinity)
	if err != nil {
		log.Panic(err)
	}

	filePtr, err := os.Open(nodeIPsFilepath)
	if err != nil {
		log.Panic(err)
	}

	var nodeIPs map[string]string

	err = json.NewDecoder(filePtr).Decode(&nodeIPs)
	if err != nil {
		log.Panic(err)
	}

	for nodeID, node := range locationsInVicinity.Nodes {
		node.Addr = nodeIPs[nodeID]
	}

	log.Debugf("Loaded IPs: %+v", locationsInVicinity.Nodes)

	return locationsInVicinity
}

// Change this for lower API call.
func registerMetricInLowerAPI(_ string) {
}
