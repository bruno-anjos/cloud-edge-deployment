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

	for metricId, metricValue := range metrics {
		log.Debugf("loaded metric %s with value %v", metricId, metricValue)
		e.TrackMetric(metricId)
		e.SetMetric(metricId, metricValue)
	}
}

func (e *Environment) TrackMetric(metricId string) {
	_, loaded := e.trackedMetrics.LoadOrStore(metricId, nil)
	if loaded {
		return
	}

	registerMetricInLowerApi(metricId)
}

func (e *Environment) GetMetric(metricId string) (value interface{}, ok bool) {
	return e.metrics.Load(metricId)
}

func (e *Environment) SetMetric(metricId string, value interface{}) {
	e.metrics.Store(metricId, value)
}

func (e *Environment) DeleteMetric(metricId string) {
	e.metrics.Delete(metricId)
}

func (e *Environment) Copy() (copy *Environment) {
	newMap := &sync.Map{}
	copy = &Environment{metrics: newMap}

	e.metrics.Range(func(key, value interface{}) bool {
		newMap.Store(key, value)
		return true
	})

	return
}

// TODO change this for lower API call
func registerMetricInLowerApi(_ string) {

}
