package environment

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
	client "github.com/nm-morais/demmon-client/pkg"
	"github.com/nm-morais/demmon-common/body_types"
	exporter "github.com/nm-morais/demmon-exporter"
	log "github.com/sirupsen/logrus"
)

type (
	typeInterestSetsValue struct {
		ISID   int64
		Finish chan interface{}
	}

	vicinityKey   = string
	vicinityValue = *utils.Node
)

type Environment struct {
	exporter     *exporter.Exporter
	interestSets sync.Map
	vicinity     sync.Map
	autoClient   autonomic.Client
	myself       *utils.Node
	location     s2.CellID

	DemmonCli *client.DemmonClient
}

const (
	logFolder = "/exporter"

	autonomicService       = "ced-autonomic"
	exportMetricsFrequency = 5 * time.Second

	queryTimeout                  = 1 * time.Second
	customInterestSetQueryTimeout = 3 * time.Second
	exportFrequencyInterestSet    = 1 * time.Second

	locationExportFrequency = 10 * time.Second

	dialBackoff = 3 * time.Second
	dialTimeout = 5 * time.Second

	defaultBucketSize = 10
	maxRetries        = 3
	defaultTTL        = 2

	DaemonPort = 8090

	ClientRequestTimeout = 5 * time.Second
	connectTimeout       = 5 * time.Second
)

func NewEnvironment(myself *utils.Node, location s2.CellID, autoClient autonomic.Client) *Environment {
	exporterConf := &exporter.Conf{
		Silent:          true,
		LogFolder:       logFolder,
		ImporterHost:    myself.Addr,
		ImporterPort:    DaemonPort,
		LogFile:         "exporter.log",
		DialAttempts:    3,
		DialBackoffTime: 1 * time.Second,
		DialTimeout:     3 * time.Second,
		RequestTimeout:  3 * time.Second,
	}

	demmonCliConf := client.DemmonClientConf{
		DemmonPort:     DaemonPort,
		DemmonHostAddr: myself.Addr,
		RequestTimeout: ClientRequestTimeout,
	}
	demmonCli := client.New(demmonCliConf)
	err := demmonCli.ConnectTimeout(connectTimeout)
	if err != nil {
		log.Panic(err)
	}

	exp, err := exporter.New(exporterConf, myself.Addr, autonomicService, nil)
	if err != nil {
		log.Panic(err)
	}

	return &Environment{
		exporter:     exp,
		interestSets: sync.Map{},
		vicinity:     sync.Map{},
		autoClient:   autoClient,
		myself:       myself,
		location:     location,
		DemmonCli:    demmonCli,
	}
}

func installNeighborLocationQuery(demmonCli *client.DemmonClient) {
	locationNeighborQuery := fmt.Sprintf(`SelectLast("%s","*")`, MetricLocation)
	_, err := demmonCli.InstallNeighborhoodInterestSet(&body_types.NeighborhoodInterestSet{
		IS: body_types.InterestSet{
			MaxRetries: maxRetries,
			Query: body_types.RunnableExpression{
				Expression: locationNeighborQuery,
				Timeout:    customInterestSetQueryTimeout,
			},
			OutputBucketOpts: body_types.BucketOptions{
				Name: MetricLocationInVicinity,
				Granularity: body_types.Granularity{
					Granularity: locationExportFrequency / 2,
					Count:       defaultBucketSize,
				},
			},
		},
		TTL: defaultTTL,
	})
	if err != nil {
		log.Panic(err)
	}
}

func exportDefaults(demmonCli *client.DemmonClient, exp *exporter.Exporter, myself *utils.Node, location s2.CellID) {
	installedChan := make(chan interface{})

	go exportLocationPeriodically(demmonCli, myself, location, installedChan)
	go startExporting(exp, installedChan)
}

func startExporting(exp *exporter.Exporter, installedChan <-chan interface{}) {
	<-installedChan
	exp.ExportLoop(context.Background(), exportMetricsFrequency)
}

func exportLocationPeriodically(demmonCli *client.DemmonClient, myself *utils.Node, location s2.CellID,
	installedChan chan<- interface{}) {
	ticker := time.NewTicker(locationExportFrequency)
	err := demmonCli.InstallBucket(MetricLocation, locationExportFrequency, defaultBucketSize)
	if err != nil {
		log.Panic(err)
	}

	close(installedChan)

	for {
		err = demmonCli.PushMetricBlob([]body_types.TimeseriesDTO{
			{
				MeasurementName: MetricLocation,
				TSTags:          map[string]string{nodeIDTag: myself.ID},
				Values: []body_types.ObservableDTO{
					{
						TS:     time.Now(),
						Fields: map[string]interface{}{"value": location.ToToken()},
					},
				},
			},
		})
		if err != nil {
			log.Panic(err)
		}

		log.Debugf("exported location %s", location.ToToken())

		<-ticker.C
	}
}

func (e *Environment) handleNodeUpdates(updateChan <-chan body_types.NodeUpdates) {
	for nodeUpdate := range updateChan {
		log.Debugf("node update: %+v\n", nodeUpdate)
		addr := nodeUpdate.Peer.IP.String() + ":" + strconv.Itoa(autonomic.Port)

		switch nodeUpdate.Type {
		case body_types.NodeDown:
			id, status := e.autoClient.GetID(addr)
			if status != http.StatusOK {
				log.Panicf("got status %d while getting id for %s", status, nodeUpdate.Peer.IP)
			}

			e.vicinity.Delete(id)
		case body_types.NodeUp:
			id, status := e.autoClient.GetID(addr)
			if status != http.StatusOK {
				log.Panicf("got status %d while getting id for %s", status, nodeUpdate.Peer.IP)
			}

			node := &utils.Node{
				ID:   id,
				Addr: nodeUpdate.Peer.IP.String(),
			}

			e.vicinity.Store(id, node)
		}
	}
}

func (e *Environment) AddDeploymentInterestSet(deploymentID, query, outputMetricID string) {
	auxID := getDeploymentIDMetricString(deploymentID, outputMetricID)

	ISID, errChan, finishChan, err := e.DemmonCli.InstallCustomInterestSet(body_types.CustomInterestSet{
		DialRetryBackoff: dialBackoff,
		DialTimeout:      dialTimeout,
		Hosts:            []body_types.CustomInterestSetHost{},
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

func (e *Environment) UpdateDeploymentInterestSet(deploymentID, outputMetricID string, hosts []*utils.Node) {
	auxID := getDeploymentIDMetricString(deploymentID, outputMetricID)

	if value, ok := e.interestSets.Load(auxID); !ok {
		log.Warn("no interest set for deployment %s - %s", deploymentID, outputMetricID)

		return
	} else {
		ISHosts := make([]body_types.CustomInterestSetHost, len(hosts))
		for i, host := range hosts {
			ISHosts[i] = body_types.CustomInterestSetHost{
				IP:   net.ParseIP(host.Addr),
				Port: DaemonPort,
			}
		}

		ISvalue := value.(*typeInterestSetsValue)

		err := e.DemmonCli.UpdateCustomInterestSet(body_types.UpdateCustomInterestSetReq{
			SetID: ISvalue.ISID,
			Hosts: ISHosts,
		})
		if err != nil {
			log.Panic(err)
		}
	}
}

func (e *Environment) GetVicinity() (vicinity map[string]*utils.Node) {
	vicinity = map[string]*utils.Node{}
	e.vicinity.Range(func(key, value interface{}) bool {
		id := key.(vicinityKey)
		node := value.(vicinityValue)

		vicinity[id] = node

		return true
	})

	return
}

func (e *Environment) GetLocationInVicinity() map[string]s2.CellID {
	query := fmt.Sprintf(LocationQuery, MetricLocationInVicinity)

	timeseries, err := e.DemmonCli.Query(query, queryTimeout)
	if err != nil {
		log.Panic(err)
	}

	locations := map[string]s2.CellID{}
	for _, ts := range timeseries {
		hostID := ts.TSTags[nodeIDTag]
		locations[hostID] = s2.CellIDFromToken(ts.Values[0].Fields["value"].(string))
	}

	log.Debugf("Got locations in vicinity: %+v, %+v", locations, timeseries)

	return locations
}

func (e *Environment) IsInVicinity(nodeID string) bool {
	_, ok := e.vicinity.Load(nodeID)
	return ok
}

func (e *Environment) Start() {
	exportDefaults(e.DemmonCli, e.exporter, e.myself, e.location)

	res, err, _, updateChan := e.DemmonCli.SubscribeNodeUpdates()
	if err != nil {
		log.Panic(err)
	}

	log.Debugf("Starting view: %+v", res)

	installNeighborLocationQuery(e.DemmonCli)

	go e.handleNodeUpdates(updateChan)
}

func getDeploymentIDMetricString(deploymentID, metricID string) string {
	return fmt.Sprintf("%s-%s", deploymentID, metricID)
}

const (
	DeploymentTag = "Deployment"

	locationWithHostQuery  = `SelectLast("%s", {"Host": "%s"})`
	LocationQuery          = `SelectLast("%s", "*")`
	loadPerDeploymentQuery = `Avg(Select("%s", {"Host": "%s", "Deployment": "%s"}), "value")`
)

func GetLocation(demmonCli *client.DemmonClient, host *utils.Node) s2.CellID {
	query := fmt.Sprintf(locationWithHostQuery, MetricLocation, host.Addr)

	timeseries, err := demmonCli.Query(query, queryTimeout)
	if err != nil {
		log.Panic(err)
	}

	return s2.CellIDFromToken(timeseries[0].Values[0].Fields["value"].(string))
}

func GetLoad(demmonCli *client.DemmonClient, deploymentID string, host *utils.Node) int {
	query := fmt.Sprintf(loadPerDeploymentQuery, MetricLoad, deploymentID, host.Addr)

	timeseries, err := demmonCli.Query(query, queryTimeout)
	if err != nil {
		log.Warn(err)
		return 0
	}

	if len(timeseries) > 0 && len(timeseries[0].Values) > 0 {
		return int(timeseries[0].Values[0].Fields["avg_value"].(float64))
	}

	return 0
}
