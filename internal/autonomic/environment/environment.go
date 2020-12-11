package environment

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
)

type Environment struct {
	trackedMetrics *sync.Map
	metrics        *sync.Map
}

const (
	metricsFolder        = "metrics/"
	metricsFileExtension = ".met"
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

	var metrics map[string]interface{}

	err = json.Unmarshal(data, &metrics)
	if err != nil {
		panic(err)
	}

	for metricID, metricValue := range metrics {
		log.Debugf("loaded metric %s with value %v", metricID, metricValue)
		e.TrackMetric(metricID)
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

// Change this for lower API call.
func registerMetricInLowerAPI(_ string) {
}
