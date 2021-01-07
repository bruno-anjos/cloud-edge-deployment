package environment

import (
	"context"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	exporter "github.com/nm-morais/demmon-exporter"
)

type Environment struct {
	exporter *exporter.Exporter
}

const (
	logFolder = "/exporter"

	autonomicService       = "ced-autonomic"
	exportMetricsFrequency = 5 * time.Second
)

var exporterConf = &exporter.Conf{
	Silent:          true,
	LogFolder:       logFolder,
	ImporterHost:    "localhost",
	ImporterPort:    8090,
	LogFile:         "exporter.log",
	DialAttempts:    3,
	DialBackoffTime: 1 * time.Second,
	DialTimeout:     3 * time.Second,
	RequestTimeout:  3 * time.Second,
}

func NewEnvironment(myself *utils.Node) *Environment {
	e, err := exporter.New(exporterConf, myself.Addr, autonomicService, nil)
	if err != nil {
		panic(err)
	}

	go e.ExportLoop(context.TODO(), exportMetricsFrequency)

	return &Environment{
		exporter: e,
	}
}

func (e *Environment) GetMetric(metricID string) (value interface{}, ok bool) {
}

func (e *Environment) SetMetric(metricID string, value interface{}) {
}

func (e *Environment) DeleteMetric(metricID string) {
}
