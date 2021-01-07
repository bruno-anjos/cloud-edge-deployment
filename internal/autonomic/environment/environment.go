package environment

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
	client "github.com/nm-morais/demmon-client/pkg"
	"github.com/nm-morais/demmon-common/body_types"
	exporter "github.com/nm-morais/demmon-exporter"
	log "github.com/sirupsen/logrus"
)

type (
	typeInterestSetsValue struct {
		ISID   uint64
		Finish chan interface{}
	}
)

type Environment struct {
	exporter     *exporter.Exporter
	demmonCli    client.DemmonClient
	interestSets sync.Map
}

const (
	logFolder = "/exporter"

	autonomicService       = "ced-autonomic"
	exportMetricsFrequency = 5 * time.Second

	queryTimeout                  = 1 * time.Second
	customInterestSetQueryTimeout = 3 * time.Second
	exportFrequencyInterestSet    = 1 * time.Second

	defaultBucketSize = 10

	maxRetries = 3

	daemonPort = 8090
)

var exporterConf = &exporter.Conf{
	Silent:          true,
	LogFolder:       logFolder,
	ImporterHost:    "localhost",
	ImporterPort:    daemonPort,
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
		exporter:  e,
		demmonCli: client.DemmonClient{},
	}
}

func (e *Environment) UpdateDeploymentInterestSet(deploymentID, query, outputMetricID string, hosts []*utils.Node) {
	auxID := getDeploymentIDMetricString(deploymentID, outputMetricID)
	if value, ok := e.interestSets.Load(auxID); ok {
		ISvalue := value.(*typeInterestSetsValue)
		close(ISvalue.Finish)
		_, err := e.demmonCli.RemoveCustomInterestSet(ISvalue.ISID)
		if err != nil {
			log.Panic(err)
		}
	}

	ISHosts := make([]body_types.CustomInterestSetHost, len(hosts))
	for i, host := range hosts {
		ISHosts[i] = body_types.CustomInterestSetHost{
			IP:   net.ParseIP(host.Addr),
			Port: daemonPort,
		}
	}

	ISID, errChan, finishChan, err := e.demmonCli.InstallCustomInterestSet(body_types.CustomInterestSet{
		Hosts: []body_types.CustomInterestSetHost{},
		IS: body_types.InterestSet{
			MaxRetries: maxRetries,
			Query: body_types.RunnableExpression{
				Timeout:    customInterestSetQueryTimeout,
				Expression: query,
			},
			OutputBucketOpts: body_types.BucketOptions{
				Name: outputMetricID,
				Granularity: body_types.Granularity{
					Granularity: exportFrequencyInterestSet,
					Count:       defaultBucketSize,
				},
			},
		},
	})
	if err != nil {
		log.Panic(err)
	}

	log.Debugf("added custom interest set for deployment %s with query %s -> %s", deploymentID, query,
		outputMetricID)

	go func() {
		ISErr := <-errChan
		log.Panic("interest set: %s, err: %s", deploymentID, ISErr)
	}()

	e.interestSets.Store(auxID, &typeInterestSetsValue{
		ISID:   ISID,
		Finish: finishChan,
	})
}

func getDeploymentIDMetricString(deploymentID, metricID string) string {
	return fmt.Sprintf("%s-%s", deploymentID, metricID)
}

const (
	locationQuery           = `SelectLast(%s, {"Host": "%s"})`
	locationInVicinityQuery = `SelectLast(%s)`
	loadPerDeploymentQuery  = `Avg(Select("%s", {"Host": "%s", "Deployment": "%s"}), "value")`
)

func GetLocation(demmonCli *client.DemmonClient, host *utils.Node) s2.CellID {
	query := fmt.Sprintf(locationQuery, metricLocation, host.Addr)

	timeseries, err := demmonCli.Query(query, queryTimeout)
	if err != nil {
		log.Panic(err)
	}

	return s2.CellIDFromToken(timeseries[0].Values[0].Fields["value"].(string))
}

func GetLocationInVicinity(demmonCli *client.DemmonClient) map[string]s2.CellID {
	query := fmt.Sprintf(locationInVicinityQuery, metricLocationInVicinity)

	timeseries, err := demmonCli.Query(query, queryTimeout)
	if err != nil {
		log.Panic(err)
	}

	locations := map[string]s2.CellID{}
	for _, ts := range timeseries {
		hostID := ts.TSTags["NodeID"]
		locations[hostID] = s2.CellIDFromToken(ts.Values[0].Fields["value"].(string))
	}

	return locations
}

func GetLoad(demmonCli *client.DemmonClient, deploymentID string, host *utils.Node) float64 {
	query := fmt.Sprintf(loadPerDeploymentQuery, metricLoad, deploymentID, host.Addr)

	timeseries, err := demmonCli.Query(query, queryTimeout)
	if err != nil {
		log.Panic(err)
	}

	return timeseries[0].Values[0].Fields["avg_value"].(float64)
}
