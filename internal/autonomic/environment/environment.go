package environment

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	demmonAPI "github.com/bruno-anjos/cloud-edge-deployment/api/demmon"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
	"github.com/mitchellh/mapstructure"
	client "github.com/nm-morais/demmon-client/pkg"
	"github.com/nm-morais/demmon-common/body_types"
	exporter "github.com/nm-morais/demmon-exporter"
	log "github.com/sirupsen/logrus"
)

type (
	typeInterestSetsValue struct {
		ISID   *string
		Finish chan interface{}
	}

	nodeWithLocation struct {
		Node     *utils.Node
		Location string
	}

	vicinityKey   = string
	vicinityValue = *utils.Node
)

type Environment struct {
	exporter              *exporter.Exporter
	interestSets          sync.Map
	vicinity              sync.Map
	vicinityInterestSetID *string
	autoClient            autonomic.Client
	myself                *utils.Node
	location              s2.CellID

	DemmonCli *client.DemmonClient
}

const (
	logFolder = "/exporter"

	autonomicService       = "ced-autonomic"
	exportMetricsFrequency = 5 * time.Second

	queryTimeout                  = 1 * time.Second
	customInterestSetQueryTimeout = 3 * time.Second
	exportFrequencyInterestSet    = 10 * time.Second

	locationExportFrequency = 10 * time.Second

	dialBackoff = 3 * time.Second
	dialTimeout = 5 * time.Second

	defaultBucketSize = 10
	maxRetries        = 3

	DaemonPort = 8090

	ClientRequestTimeout = 5 * time.Second
	connectTimeout       = 5 * time.Second

	horizonDistance = 3
)

var knownLocations = sync.Map{}

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

	connectErrChan := RetryConnect(demmonCli)

	go internalUtils.PanicOnErrFromChan(connectErrChan)

	exp, err, expErrChan := exporter.New(exporterConf, myself.Addr, autonomicService, nil)
	if err != nil {
		log.Panic(err)
	}

	go internalUtils.PanicOnErrFromChan(expErrChan)

	return &Environment{
		exporter:              exp,
		interestSets:          sync.Map{},
		vicinity:              sync.Map{},
		vicinityInterestSetID: nil,
		autoClient:            autoClient,
		myself:                myself,
		location:              location,
		DemmonCli:             demmonCli,
	}
}

func (e *Environment) installNeighborLocationQuery(demmonCli *client.DemmonClient) {
	locationNeighborQuery := fmt.Sprintf(`SelectLast("%s","*")`, MetricLocation)

	set := body_types.CustomInterestSet{
		DialRetryBackoff: dialBackoff,
		DialTimeout:      dialTimeout,
		Hosts:            []body_types.CustomInterestSetHost{},
		IS: body_types.InterestSet{
			MaxRetries: maxRetries,
			Query: body_types.RunnableExpression{
				Expression: locationNeighborQuery,
				Timeout:    customInterestSetQueryTimeout,
			},
			OutputBucketOpts: body_types.BucketOptions{
				Name: MetricLocationInVicinity,
				Granularity: body_types.Granularity{
					Granularity: locationExportFrequency,
					Count:       defaultBucketSize,
				},
			},
		},
	}

	ISID, _, errChan := retryInstallCustomSet(demmonCli, set)

	go internalUtils.PanicOnErrFromChan(errChan)

	log.Debugf("got vicinity interest set %d", ISID)

	e.vicinityInterestSetID = ISID
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

	var (
		err   error
		retry = true
	)

	for retry {
		err = demmonCli.InstallBucket(MetricLocation, locationExportFrequency, defaultBucketSize)
		if err != nil {
			if !strings.Contains(err.Error(), errTimedOut) {
				log.Panic(err)
			}
		} else {
			retry = false
		}
	}

	close(installedChan)

	for {
		timeseriesDTO := []body_types.TimeseriesDTO{
			{
				MeasurementName: MetricLocation,
				TSTags: map[string]string{
					nodeIDTag: myself.ID,
					hostTag:   myself.Addr,
				},
				Values: []body_types.ObservableDTO{
					{
						TS:     time.Now(),
						Fields: map[string]interface{}{"value": location.ToToken()},
					},
				},
			},
		}
		retryPushMetric(demmonCli, timeseriesDTO)

		log.Debugf("exported location %s", location.ToToken())

		<-ticker.C
	}
}

func SetupClientCentroidsExport(demmCli *client.DemmonClient) {
	err := demmCli.InstallBucket(MetricCentroids, locationExportFrequency, defaultBucketSize)
	if err != nil {
		log.Panic(err)
	}
}

func (e *Environment) AddDeploymentInterestSet(deploymentID, query, outputMetricID string) {
	auxID := getDeploymentIDMetricString(deploymentID, outputMetricID)

	set := body_types.CustomInterestSet{
		DialRetryBackoff: dialBackoff,
		DialTimeout:      dialTimeout,
		Hosts:            []body_types.CustomInterestSetHost{},
		IS: body_types.InterestSet{
			MaxRetries: maxRetries,
			Query: body_types.RunnableExpression{
				Timeout:    customInterestSetQueryTimeout,
				Expression: fmt.Sprintf(query, outputMetricID),
			},
			OutputBucketOpts: body_types.BucketOptions{
				Name: outputMetricID,
				Granularity: body_types.Granularity{
					Granularity: exportFrequencyInterestSet,
					Count:       defaultBucketSize,
				},
			},
		},
	}

	ISID, finishChan, errChan := retryInstallCustomSet(e.DemmonCli, set)

	log.Debugf("added custom interest set for deployment %s with query %s -> %s", deploymentID, query,
		outputMetricID)

	go func() {
		ISErr := <-errChan
		log.Panicf("interest set: %s, err: %s", deploymentID, ISErr)
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

		var (
			err    error
			setReq = body_types.UpdateCustomInterestSetReq{
				SetID: *ISvalue.ISID,
				Hosts: ISHosts,
			}
			retry = true
		)

		for retry {
			err = e.DemmonCli.UpdateCustomInterestSet(setReq)
			if err != nil {
				if !strings.Contains(err.Error(), errTimedOut) {
					log.Error(err)
					retry = false
				}
			} else {
				retry = true
			}
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

	timeseries := retryQuery(e.DemmonCli, query)

	locations := map[string]s2.CellID{}
	for _, ts := range timeseries {
		hostID := ts.TSTags[nodeIDTag]
		locations[hostID] = s2.CellIDFromToken(ts.Values[0].Fields["value"].(string))
	}

	log.Debugf("Got locations in vicinity: %+v", locations)

	return locations
}

func (e *Environment) IsInVicinity(nodeID string) bool {
	_, ok := e.vicinity.Load(nodeID)
	return ok
}

func (e *Environment) Start() {
	exportDefaults(e.DemmonCli, e.exporter, e.myself, e.location)

	e.installNeighborLocationQuery(e.DemmonCli)

	go e.announceMyselfPeriodically()
	go e.handleAnnouncements()
}

const (
	announcementTimeout = 10 * time.Second
)

func (e *Environment) announceMyselfPeriodically() {
	for {
		log.Debugf("announcing %+v", e.myself)

		var (
			err   error
			retry = true
			msg   = body_types.Message{
				ID:  demmonAPI.NodeAnnouncement,
				TTL: horizonDistance,
				Content: &nodeWithLocation{
					Node:     e.myself,
					Location: e.location.ToToken(),
				},
			}
		)

		for retry {
			err = e.DemmonCli.BroadcastMessage(msg)
			if err != nil {
				if !strings.Contains(err.Error(), errTimedOut) {
					log.Panic(err)
				}
			} else {
				retry = false
			}
		}

		time.Sleep(announcementTimeout)
	}
}

func (e *Environment) handleAnnouncements() {
	var (
		err     error
		msgChan chan body_types.Message
		retry   = true
	)

	for retry {
		msgChan, _, err = e.DemmonCli.InstallBroadcastMessageHandler(demmonAPI.NodeAnnouncement)
		if err != nil {
			if !strings.Contains(err.Error(), errTimedOut) {
				log.Panic(err)
			}
		} else {
			retry = false
		}
	}

	log.Debugf("started handling announcements...")

	nodesAlive := &sync.Map{}
	go e.handleCleaningStaleNodes(nodesAlive)

	for msg := range msgChan {
		log.Debugf("got msg %+v", msg)
		switch msg.ID {
		case demmonAPI.NodeAnnouncement:
			var nodeAnnounced nodeWithLocation

			err = mapstructure.Decode(msg.Content, &nodeAnnounced)
			if err != nil {
				log.Panic(err)
			}

			log.Debugf("adding neighbor %s", nodeAnnounced.Node.ID)

			nodesAlive.Store(nodeAnnounced.Node.ID, nil)
			e.vicinity.Store(nodeAnnounced.Node.ID, nodeAnnounced.Node)
			knownLocations.Store(nodeAnnounced.Node.ID,
				s2.CellIDFromToken(nodeAnnounced.Location))
			e.updateVicinityInterestSet()

			log.Debug("handled node announcement")
		}
	}
}

func (e *Environment) updateVicinityInterestSet() {
	var hosts []body_types.CustomInterestSetHost

	e.vicinity.Range(func(key, value interface{}) bool {
		node := value.(*utils.Node)

		hosts = append(hosts, body_types.CustomInterestSetHost{
			IP:   net.ParseIP(node.Addr),
			Port: DaemonPort,
		})

		return true
	})

	log.Debug("updating interest set")

	var (
		err   error
		retry = true
		set   = body_types.UpdateCustomInterestSetReq{
			SetID: *e.vicinityInterestSetID,
			Hosts: hosts,
		}
	)

	for retry {
		err = e.DemmonCli.UpdateCustomInterestSet(set)
		if err != nil {
			if !strings.Contains(err.Error(), errTimedOut) {
				log.Error(err)
				retry = false
			}
		} else {
			retry = false
		}
	}

	log.Debug("updated interest set")
}

func (e *Environment) handleCleaningStaleNodes(nodesAlive *sync.Map) {
	suspectedNodes := map[string]interface{}{}

	for {
		nodesAlive.Range(func(key, _ interface{}) bool {
			nodeID := key.(string)

			suspectedNodes[nodeID] = nil

			nodesAlive.Delete(nodeID)

			return true
		})

		time.Sleep(announcementTimeout * 2)

		for nodeID := range suspectedNodes {
			if _, ok := nodesAlive.Load(nodeID); !ok {
				log.Debugf("removing stale neighbor %s", nodeID)
				e.vicinity.Delete(nodeID)
			}
		}
	}
}

func getDeploymentIDMetricString(deploymentID, metricID string) string {
	return fmt.Sprintf("%s-%s", deploymentID, metricID)
}

const (
	DeploymentTag = "deployment"

	locationWithHostQuery  = `SelectLast("%s", {"host": "%s"})`
	LocationQuery          = `SelectLast("%s", "*")`
	loadPerDeploymentQuery = `Avg(Select("%s", {"host": "%s", "deployment": "%s"}), "value")`
	centroidsQuery         = `SelectLast("%s", {"host": "%s", "deployment": "%s"})`

	errTimedOut = "timed out"
)

func GetLocation(demmonCli *client.DemmonClient, host *utils.Node) s2.CellID {
	query := fmt.Sprintf(locationWithHostQuery, MetricLocationInVicinity, host.Addr)

	timeseries := retryQuery(demmonCli, query)

	if len(timeseries) == 0 || len(timeseries[0].Values) == 0 {
		value, ok := knownLocations.Load(host.ID)
		if !ok {
			log.Panic("no location for %s", host.ID)
		}

		return value.(s2.CellID)
	}

	return s2.CellIDFromToken(timeseries[0].Values[0].Fields["value"].(string))
}

func GetLoad(demmonCli *client.DemmonClient, deploymentID string, host *utils.Node) int {
	query := fmt.Sprintf(loadPerDeploymentQuery, MetricLoad, host.Addr, deploymentID)

	timeseries := retryQuery(demmonCli, query)

	if len(timeseries) > 0 && len(timeseries[0].Values) > 0 {
		return int(timeseries[0].Values[0].Fields["avg_value"].(float64))
	}

	return 0
}

func ExportClientCentroids(demmonCli *client.DemmonClient, deploymentID string, node *utils.Node,
	centroids []s2.CellID) {
	centroidTokens := make([]string, len(centroids))
	for i, centroid := range centroids {
		centroidTokens[i] = centroid.ToToken()
	}

	timeseriesDTO := []body_types.TimeseriesDTO{
		{
			MeasurementName: MetricCentroids,
			TSTags:          map[string]string{hostTag: node.Addr, DeploymentTag: deploymentID},
			Values: []body_types.ObservableDTO{
				{
					TS:     time.Now(),
					Fields: map[string]interface{}{"value": centroidTokens},
				},
			},
		},
	}

	retryPushMetric(demmonCli, timeseriesDTO)
}

func GetClientCentroids(demmonCli *client.DemmonClient, deploymentID string, node *utils.Node) []s2.CellID {
	query := fmt.Sprintf(centroidsQuery, MetricCentroids, node.Addr, deploymentID)

	timeseries := retryQuery(demmonCli, query)

	if len(timeseries) == 0 || len(timeseries[0].Values) == 0 {
		return nil
	}

	values := timeseries[0].Values[0].Fields["value"].([]interface{})
	centroids := make([]s2.CellID, len(values))

	for i := range values {
		centroids[i] = s2.CellIDFromToken(values[i].(string))
	}

	return centroids
}

func retryQuery(demmonCli *client.DemmonClient, query string) (timeseries []body_types.TimeseriesDTO) {
	var (
		err   error
		retry = true
	)

	for retry {
		timeseries, err = demmonCli.Query(query, queryTimeout)
		if err != nil {
			if !strings.Contains(err.Error(), errTimedOut) {
				log.Warn(err)
				retry = false
			}
		} else {
			retry = false
		}
	}

	return
}

func retryPushMetric(demmonCli *client.DemmonClient, timeseriesDTO []body_types.TimeseriesDTO) {
	var (
		retry = true
		err   error
	)

	for retry {
		err = demmonCli.PushMetricBlob(timeseriesDTO)
		if err != nil {
			if !strings.Contains(err.Error(), errTimedOut) {
				log.Error(err)
				retry = false
			}
		} else {
			retry = false
		}
	}
}

func retryInstallCustomSet(demmonCli *client.DemmonClient, set body_types.CustomInterestSet) (ISID *string, finishChan chan interface{}, errChan chan error) {
	retry := true

	var err error

	for retry {
		ISID, errChan, finishChan, err = demmonCli.InstallCustomInterestSet(set)

		if err != nil {
			if !strings.Contains(err.Error(), errTimedOut) {
				log.Panic(err)
			}
		} else {
			retry = false
		}
	}

	return
}

func RetryConnect(demmonCli *client.DemmonClient) (connectErrChan chan error) {
	var (
		retry = true
		err   error
	)

	for retry {
		err, connectErrChan = demmonCli.ConnectTimeout(connectTimeout)
		if err != nil {
			if !strings.Contains(err.Error(), errTimedOut) {
				log.Panic(err)
			}
		} else {
			retry = false
		}
	}

	return
}
