package clients

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	archimedesHTTPClient "github.com/bruno-anjos/archimedesHTTPClient"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/archimedes/cell"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/deployment"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/environment"
	internalUtils "github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"
	"github.com/golang/geo/s2"
	client "github.com/nm-morais/demmon-client/pkg"
	"github.com/nm-morais/demmon-common/exporters"
	exporter "github.com/nm-morais/demmon-exporter"
	log "github.com/sirupsen/logrus"
)

type (
	batchValue struct {
		Locations *sync.Map
		NumReqs   int64
	}

	exploringCellsValueType = []s2.CellID

	currBatchMapKey   = string
	currBatchMapValue = *batchValue

	numReqsLastMinuteMapValue = *batchValue

	gaugesMapKey   = string
	gaugesMapValue = exporters.Gauge

	Manager struct {
		numReqsLastMinute sync.Map
		currBatch         sync.Map
		exploringCells    sync.Map
		gauges            sync.Map
		cellManager       *cell.Manager
		gauge             *exporter.Gauge
	}
)

const (
	batchTimer             = 5 * time.Second
	loadExportFrequency    = 5 * time.Second
	metricsExportFrequency = 20 * time.Second

	archimedesService = "ced-archimedes"

	logFolder  = "/exporter"
	daemonPort = 8090

	loadSamples = 10
)

func NewManager(demmCli *client.DemmonClient, myself *utils.Node) *Manager {
	exporterConf := &exporter.Conf{
		Silent:          true,
		LogFolder:       logFolder,
		ImporterHost:    deployment.Myself.Addr,
		ImporterPort:    daemonPort,
		LogFile:         "exporter.log",
		DialAttempts:    3,
		DialBackoffTime: 1 * time.Second,
		DialTimeout:     3 * time.Second,
		RequestTimeout:  3 * time.Second,
	}

	exp, err, errChan := exporter.New(exporterConf, deployment.Myself.Addr, archimedesService, nil)
	if err != nil {
		log.Panic(err)
	}

	go internalUtils.PanicOnErrFromChan(errChan)

	r := &Manager{
		numReqsLastMinute: sync.Map{},
		currBatch:         sync.Map{},
		exploringCells:    sync.Map{},
		gauges:            sync.Map{},
		cellManager:       cell.NewManager(demmCli, myself),
		gauge:             exp.NewGauge(environment.MetricLoad, loadSamples),
	}

	go exp.ExportLoop(context.Background(), metricsExportFrequency)

	go r.exportLoadsPeriodically()
	go r.manageLoadBatch()

	return r
}

func (r *Manager) exportLoadsPeriodically() {
	ticker := time.NewTicker(loadExportFrequency)
	for {
		r.gauges.Range(func(key, value interface{}) bool {
			deploymentID := key.(gaugesMapKey)
			gauge := value.(gaugesMapValue)

			load := r.getLoad(deploymentID)
			log.Debugf("exporting load %d for deployment %s", load, deploymentID)

			gauge.Set(float64(load))

			return true
		})

		<-ticker.C
	}
}

func (r *Manager) AddDeployment(deploymentID string) {
	reqsLastMinute := &batchValue{
		Locations: &sync.Map{},
		NumReqs:   0,
	}
	r.numReqsLastMinute.LoadOrStore(deploymentID, reqsLastMinute)

	newBatch := &batchValue{
		Locations: &sync.Map{},
		NumReqs:   0,
	}
	r.currBatch.LoadOrStore(deploymentID, newBatch)

	gauge := r.gauge.With(environment.DeploymentTag, deploymentID)
	r.gauges.LoadOrStore(deploymentID, gauge)

	log.Debugf("registered gauge for %s", deploymentID)
}

func (r *Manager) UpdateNumRequests(deploymentID string, location s2.CellID) {
	r.updateEntry(deploymentID, location)
	r.updateBatch(deploymentID, location)

	r.cellManager.AddClientToDownmostCell(deploymentID, location)
}

// Even though this is thread safe, this does not guarantee a perfectly accurate count
// of requests received since one can load the entry, meanwhile the entry is swapped, and
// increment the entry that is stale, thus never reflecting the count on the updated entry.
func (r *Manager) updateEntry(deploymentID string, location s2.CellID) {
	// load the entry
	value, ok := r.numReqsLastMinute.Load(deploymentID)
	if !ok {
		return
	}

	// possibly increment an entry that is already stale
	entry := value.(numReqsLastMinuteMapValue)

	intValue := new(int64)
	value, _ = entry.Locations.LoadOrStore(location, intValue)
	intValue = value.(cell.LocationsMapValue)

	atomic.AddInt64(intValue, 1)
	atomic.AddInt64(&entry.NumReqs, 1)

	log.Debugf("adding to numRequests %d (%d)", location, atomic.LoadInt64(&entry.NumReqs))
}

// Same as updateEntry
func (r *Manager) updateBatch(deploymentID string, location s2.CellID) {
	value, ok := r.currBatch.Load(deploymentID)
	if !ok {
		return
	}

	entry := value.(currBatchMapValue)

	intValue := new(int64)
	value, _ = entry.Locations.LoadOrStore(location, intValue)
	intValue = value.(cell.LocationsMapValue)

	atomic.AddInt64(intValue, 1)
	atomic.AddInt64(&entry.NumReqs, 1)

	log.Debugf("adding to batch %d (%d)", location, atomic.LoadInt64(&entry.NumReqs))
}

func (r *Manager) getLoad(deploymentID string) (load int) {
	value, ok := r.numReqsLastMinute.Load(deploymentID)
	if !ok {
		return 0
	}

	entry := value.(numReqsLastMinuteMapValue)
	load = int(entry.NumReqs)

	return
}

func (r *Manager) SetToExploring(deploymentID string, cells []s2.CellID) {
	r.exploringCells.Store(deploymentID, cells)
}

func (r *Manager) RemoveFromExploring(deploymentID string) {
	r.exploringCells.Delete(deploymentID)
}

func (r *Manager) GetDeploymentClientsCentroids(deploymentID string) (cells []s2.CellID, ok bool) {
	cells, ok = r.cellManager.GetDeploymentCentroids(deploymentID)
	if len(cells) == 0 || !ok {
		value, eOk := r.exploringCells.Load(deploymentID)
		if eOk {
			cells = value.(exploringCellsValueType)
		}

		ok = eOk
	}

	return
}

func (r *Manager) manageLoadBatch() {
	ticker := time.NewTicker(batchTimer)

	for {
		<-ticker.C

		r.currBatch.Range(func(key, value interface{}) bool {
			deploymentID := key.(currBatchMapKey)
			depBatch := value.(currBatchMapValue)

			newBatch := &batchValue{
				Locations: &sync.Map{},
				NumReqs:   0,
			}
			r.currBatch.Store(deploymentID, newBatch)
			go r.waitToRemove(deploymentID, depBatch)

			return true
		})
	}
}

func (r *Manager) waitToRemove(deploymentID string, batch *batchValue) {
	time.Sleep(archimedesHTTPClient.CacheExpiringTime)

	// load numRequests and decrement the amount of requests by the amount of requests in this batch
	value, ok := r.numReqsLastMinute.Load(deploymentID)
	if !ok {
		return
	}

	entry := value.(numReqsLastMinuteMapValue)

	atomic.AddInt64(&entry.NumReqs, -atomic.LoadInt64(&batch.NumReqs))

	// iterate this batch locations and decrement the count of each location in numRequests by the amount
	// of each location on this batch
	entry.Locations.Range(func(key, value interface{}) bool {
		locID := key.(cell.LocationsMapKey)
		amount := value.(cell.LocationsMapValue)
		value, ok = r.numReqsLastMinute.Load(deploymentID)
		if !ok {
			return false
		}

		entry = value.(numReqsLastMinuteMapValue)
		value, ok = entry.Locations.Load(locID)
		if !ok {
			return false
		}

		intValue := value.(cell.LocationsMapValue)

		atomic.AddInt64(intValue, -atomic.LoadInt64(amount))

		return true
	})

	r.cellManager.RemoveClientsFromCells(deploymentID, entry.Locations)
}
