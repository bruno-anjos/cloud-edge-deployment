package deployer

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	archimedes2 "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"
)

const (
	staticValue    = 1
	notStaticValue = 0

	orphanValue    = 1
	notOrphanValue = 0
)

type (
	hierarchyEntry struct {
		parent              *utils.Node
		grandparent         *utils.Node
		children            sync.Map
		numChildren         int32
		static              int32
		orphan              int32
		deploymentYAMLBytes []byte
		depthFactor         float64
		newParentChan       chan<- string
		*sync.RWMutex
	}
)

func (e *hierarchyEntry) getDeploymentYAMLBytes() []byte {
	// no lock since this value never changes
	return e.deploymentYAMLBytes
}

func (e *hierarchyEntry) hasParent() bool {
	e.RLock()
	e.RUnlock()
	return e.parent != nil
}

func (e *hierarchyEntry) getParent() (parent utils.Node) {
	e.RLock()
	defer e.RUnlock()

	return *e.parent
}

func (e *hierarchyEntry) setParent(node utils.Node) {
	e.Lock()
	defer e.Unlock()

	e.parent = &node

	if e.newParentChan != nil {
		e.newParentChan <- e.parent.Id
		close(e.newParentChan)
		e.newParentChan = nil
	}
}

func (e *hierarchyEntry) removeParent() {
	e.Lock()
	defer e.Unlock()

	e.parent = nil
}

func (e *hierarchyEntry) hasGrandparent() bool {
	e.RLock()
	e.RUnlock()
	return e.grandparent != nil
}

func (e *hierarchyEntry) getGrandparent() (grandparent utils.Node) {
	e.RLock()
	defer e.RUnlock()
	grandparent = *e.grandparent

	return
}

func (e *hierarchyEntry) setGrandparent(grandparent utils.Node) {
	e.Lock()
	defer e.Unlock()
	e.grandparent = &grandparent
}

func (e *hierarchyEntry) removeGrandparent() {
	e.Lock()
	defer e.Unlock()
	e.grandparent = nil
}

func (e *hierarchyEntry) addChild(child utils.Node) {
	if _, ok := e.children.Load(child.Id); ok {
		return
	}
	e.children.Store(child.Id, &child)
	atomic.AddInt32(&e.numChildren, 1)
}

func (e *hierarchyEntry) removeChild(childId string) {
	e.children.Delete(childId)
	atomic.AddInt32(&e.numChildren, -1)
}

func (e *hierarchyEntry) isStatic() bool {
	return atomic.LoadInt32(&e.static) == staticValue
}

func (e *hierarchyEntry) setStatic(static bool) {
	if static {
		atomic.StoreInt32(&e.static, staticValue)
	} else {
		atomic.StoreInt32(&e.static, notStaticValue)
	}
}

func (e *hierarchyEntry) isOrphan() bool {
	return atomic.LoadInt32(&e.orphan) == orphanValue
}

func (e *hierarchyEntry) setOrphan(isOrphan bool) chan string {
	if isOrphan {
		newParentChan := make(chan string)
		e.Lock()
		e.newParentChan = newParentChan
		e.Unlock()
		atomic.StoreInt32(&e.orphan, orphanValue)
		return newParentChan
	} else {
		atomic.StoreInt32(&e.orphan, notOrphanValue)
		return nil
	}
}

func (e *hierarchyEntry) getNumChildren() int {
	return int(atomic.LoadInt32(&e.numChildren))
}

func (e *hierarchyEntry) getChildren() map[string]utils.Node {
	entryChildren := map[string]utils.Node{}

	e.children.Range(func(key, value interface{}) bool {
		childId := key.(typeChildMapKey)
		child := value.(typeChildMapValue)
		entryChildren[childId] = *child
		return true
	})

	return entryChildren
}

func (e *hierarchyEntry) toDTO() *api.HierarchyEntryDTO {
	e.RLock()
	defer e.RUnlock()

	return &api.HierarchyEntryDTO{
		Parent:      e.parent,
		Grandparent: e.grandparent,
		Children:    e.getChildren(),
		Static:      e.static == staticValue,
		IsOrphan:    e.orphan == orphanValue,
	}
}

type (
	typeChildMapKey   = string
	typeChildMapValue = *utils.Node

	hierarchyTable struct {
		hierarchyEntries sync.Map
		sync.Mutex
	}

	typeHierarchyEntriesMapKey   = string
	typeHierarchyEntriesMapValue = *hierarchyEntry
)

func newHierarchyTable() *hierarchyTable {
	return &hierarchyTable{
		hierarchyEntries: sync.Map{},
	}
}

func (t *hierarchyTable) updateDeployment(deploymentId string, parent *utils.Node, grandparent *utils.Node) bool {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return false
	}

	entry := value.(typeHierarchyEntriesMapValue)

	parentId := "nil"
	if parent != nil {
		parentId = parent.Id
		entry.setParent(*parent)
	} else {
		entry.removeParent()
	}

	grandparentId := "nil"
	if grandparent != nil {
		grandparentId = grandparent.Id
		entry.setGrandparent(*grandparent)
	} else {
		entry.removeGrandparent()
	}

	log.Debugf("deployment %s updated with parent %s and grandparent %s", deploymentId, parentId, grandparentId)

	if parent != nil {
		autonomicClient.SetDeploymentParent(deploymentId, parent)
		deplClient := deplFactory.New(parent.Addr + ":" + strconv.Itoa(deployer.Port))
		status := deplClient.PropagateLocationToHorizon(deploymentId, myself.Id, location.ID(), 0, api.Add)
		if status != http.StatusOK {
			log.Errorf("got status %d while trying to propagate location to %s for deployment %s", status,
				parent.Id, deploymentId)
		}
	}

	return true
}

func (t *hierarchyTable) addDeployment(dto *api.DeploymentDTO, depthFactor float64, exploringTTL int) bool {
	var (
		static int32
	)
	if dto.Static {
		static = staticValue
	} else {
		static = notStaticValue
	}

	entry := &hierarchyEntry{
		parent:              dto.Parent,
		grandparent:         dto.Grandparent,
		children:            sync.Map{},
		static:              static,
		orphan:              notStaticValue,
		newParentChan:       nil,
		deploymentYAMLBytes: dto.DeploymentYAMLBytes,
		RWMutex:             &sync.RWMutex{},
	}

	parentId := "nil"
	if dto.Parent != nil {
		parentId = dto.Parent.Id
	}

	grandparentId := "nil"
	if dto.Grandparent != nil {
		grandparentId = dto.Grandparent.Id
	}

	log.Debugf("deployment %s has parent %s and grandparent %s", dto.DeploymentId, parentId, grandparentId)

	_, loaded := t.hierarchyEntries.LoadOrStore(dto.DeploymentId, entry)
	if loaded {
		return false
	}

	autonomicClient.RegisterDeployment(dto.DeploymentId, autonomic.StrategyIdealLatencyId, depthFactor, exploringTTL)
	if dto.Parent != nil {
		autonomicClient.SetDeploymentParent(dto.DeploymentId, dto.Parent)
		deplClient := deplFactory.New(dto.Parent.Addr + ":" + strconv.Itoa(deployer.Port))
		status := deplClient.PropagateLocationToHorizon(dto.DeploymentId, myself.Id, location.ID(), 0, api.Add)
		if status != http.StatusOK {
			log.Errorf("got status %d while trying to propagate location to %s for deployment %s", status,
				dto.Parent.Id, dto.DeploymentId)
		}
	}

	return true
}

func (t *hierarchyTable) removeDeployment(deploymentId string) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	if entry.getNumChildren() != 0 {
		log.Errorf("removing deployment that is still someones father")
	} else {
		status := autonomicClient.DeleteDeployment(deploymentId)
		if status != http.StatusOK {
			log.Errorf("got status code %d from autonomic while deleting %s", status, deploymentId)
			return
		}

		var instances map[string]*archimedes2.Instance
		instances, status = archimedesClient.GetDeployment(deploymentId)
		if status != http.StatusOK {
			log.Errorf("got status %d while requesting deployment %s instances", status, deploymentId)
			return
		}

		status = archimedesClient.DeleteDeployment(deploymentId)
		if status != http.StatusOK {
			log.Errorf("got status code %d from archimedes", status)
			return
		}

		for instanceId := range instances {
			status = schedulerClient.StopInstance(instanceId)
			if status != http.StatusOK {
				log.Errorf("got status code %d from scheduler", status)
				return
			}
		}

		parent := t.getParent(deploymentId)
		if parent != nil {
			deplClient := deplFactory.New(parent.Addr + ":" + strconv.Itoa(deployer.Port))
			status = deplClient.PropagateLocationToHorizon(deploymentId, myself.Id, location.ID(), 0, api.Remove)
			if status != http.StatusOK {
				log.Errorf("got status %d while propagating location to %s for deployment %s", status, parent.Id,
					deploymentId)
			}
		}

		t.hierarchyEntries.Delete(deploymentId)
	}
}

func (t *hierarchyTable) hasDeployment(deploymentId string) bool {
	_, ok := t.hierarchyEntries.Load(deploymentId)
	return ok
}

func (t *hierarchyTable) setDeploymentParent(deploymentId string, parent *utils.Node) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)

	if parent == nil {
		entry.removeParent()
	} else {
		entry.setParent(*parent)
	}

	auxChildren := t.getChildren(deploymentId)
	if len(auxChildren) > 0 {
		deplClient := deplFactory.New("")
		for _, child := range auxChildren {
			deplClient.SetHostPort(child.Addr + ":" + strconv.Itoa(deployer.Port))
			deplClient.SetGrandparent(deploymentId, parent)
		}
	}

	entry.setOrphan(false)
}

func (t *hierarchyTable) setDeploymentGrandparent(deploymentId string, grandparent *utils.Node) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)

	if grandparent == nil {
		entry.removeGrandparent()
	} else {
		entry.setGrandparent(*grandparent)
	}
}

func (t *hierarchyTable) setDeploymentAsOrphan(deploymentId string) <-chan string {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)
	return entry.setOrphan(true)
}

func (t *hierarchyTable) addChild(deploymentId string, child *utils.Node, exploring bool) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.addChild(*child)
	children.LoadOrStore(child.Id, child)
	autonomicClient.AddDeploymentChild(deploymentId, child)
	archimedesClient.AddDeploymentNode(deploymentId, child.Id, nodeLocCache.get(child), exploring)
	return
}

func (t *hierarchyTable) removeChild(deploymentId, childId string) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.removeChild(childId)
	autonomicClient.RemoveDeploymentChild(deploymentId, childId)
	archimedesClient.DeleteDeploymentNode(deploymentId, childId)

	return
}

func (t *hierarchyTable) getChildren(deploymentId string) (children map[string]utils.Node) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)
	return entry.getChildren()
}

func (t *hierarchyTable) getParent(deploymentId string) *utils.Node {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)
	if entry.hasParent() {
		parent := entry.getParent()
		return &parent
	} else {
		return nil
	}
}

func (t *hierarchyTable) deploymentToDTO(deploymentId string) (*api.DeploymentDTO, bool) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil, false
	}

	entry := value.(typeHierarchyEntriesMapValue)

	return &api.DeploymentDTO{
		DeploymentId:        deploymentId,
		Static:              entry.isStatic(),
		DeploymentYAMLBytes: entry.getDeploymentYAMLBytes(),
	}, true
}

func (t *hierarchyTable) isStatic(deploymentId string) bool {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return false
	}

	entry := value.(typeHierarchyEntriesMapValue)

	return entry.isStatic()
}

func (t *hierarchyTable) removeParent(deploymentId string) bool {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return false
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.removeParent()

	return true
}

func (t *hierarchyTable) getGrandparent(deploymentId string) *utils.Node {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)
	if entry.hasGrandparent() {
		grandparent := entry.getGrandparent()
		return &grandparent
	}
	return nil
}

func (t *hierarchyTable) removeGrandparent(deploymentId string) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.removeGrandparent()
}

func (t *hierarchyTable) getDeployments() []string {
	var deploymentIds []string

	t.hierarchyEntries.Range(func(key, value interface{}) bool {
		deploymentId := key.(typeHierarchyEntriesMapKey)
		deploymentIds = append(deploymentIds, deploymentId)
		return true
	})

	return deploymentIds
}

func (t *hierarchyTable) getDeploymentConfig(deploymentId string) []byte {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)
	return entry.getDeploymentYAMLBytes()
}

func (t *hierarchyTable) getDeploymentsWithParent(parentId string) (deploymentIds []string) {
	t.hierarchyEntries.Range(func(key, value interface{}) bool {
		deploymentId := key.(typeHierarchyEntriesMapKey)
		d := value.(typeHierarchyEntriesMapValue)

		if d.hasParent() && d.getParent().Id == parentId {
			deploymentIds = append(deploymentIds, deploymentId)
		}

		return true
	})

	return
}

func (t *hierarchyTable) toDTO() map[string]*api.HierarchyEntryDTO {
	entries := map[string]*api.HierarchyEntryDTO{}

	t.hierarchyEntries.Range(func(key, value interface{}) bool {
		deploymentId := key.(typeHierarchyEntriesMapKey)
		entry := value.(typeHierarchyEntriesMapValue)
		entries[deploymentId] = entry.toDTO()
		return true
	})

	return entries
}

const (
	alive     = 1
	suspected = 0
)

type (
	parentsEntry struct {
		Parent           *utils.Node
		NumOfDeployments int32
		IsUp             int32
	}

	parentsTable struct {
		parentEntries sync.Map
	}

	typeParentEntriesMapValue = *parentsEntry
)

func newParentsTable() *parentsTable {
	return &parentsTable{
		parentEntries: sync.Map{},
	}
}

func (t *parentsTable) addParent(parent *utils.Node) {
	parentEntry := &parentsEntry{
		Parent:           parent,
		NumOfDeployments: 1,
		IsUp:             alive,
	}

	log.Debugf("adding parent %s", parent.Id)

	t.parentEntries.Store(parent.Id, parentEntry)
}

func (t *parentsTable) hasParent(parentId string) bool {
	_, ok := t.parentEntries.Load(parentId)
	return ok
}

func (t *parentsTable) decreaseParentCount(parentId string) {
	value, ok := t.parentEntries.Load(parentId)
	if !ok {
		return
	}

	parentEntry := value.(typeParentEntriesMapValue)
	isZero := atomic.CompareAndSwapInt32(&parentEntry.NumOfDeployments, 1, 0)
	if isZero {
		t.removeParent(parentId)
	} else {
		atomic.AddInt32(&parentEntry.NumOfDeployments, -1)
	}

	return
}

func (t *parentsTable) removeParent(parentId string) {
	log.Debugf("removing parent %s", parentId)
	t.parentEntries.Delete(parentId)
}

func (t *parentsTable) setParentUp(parentId string) {
	value, ok := t.parentEntries.Load(parentId)
	if !ok {
		return
	}

	atomic.StoreInt32(&value.(typeParentEntriesMapValue).IsUp, alive)
}

func (t *parentsTable) checkDeadParents() (deadParents []*utils.Node) {
	t.parentEntries.Range(func(key, value interface{}) bool {
		parentEntry := value.(typeParentEntriesMapValue)

		isAlive := atomic.CompareAndSwapInt32(&parentEntry.IsUp, alive, suspected)
		log.Debugf("setting parent %s as suspected", parentEntry.Parent.Id)
		if !isAlive {
			deadParents = append(deadParents, parentEntry.Parent)
		}

		return true
	})

	return
}

/*
	HELPER METHODS
*/

func renegotiateParent(deadParent *utils.Node, alternatives map[string]*utils.Node) {
	deploymentIds := hTable.getDeploymentsWithParent(deadParent.Id)

	log.Debugf("renegotiating deployments %+v with parent %s", deploymentIds, deadParent.Id)

	for _, deploymentId := range deploymentIds {
		grandparent := hTable.getGrandparent(deploymentId)
		grandparentId := "nil"
		if grandparent != nil {
			grandparentId = grandparent.Id
		}
		log.Debugf("my grandparent for %s is %s", deploymentId, grandparentId)
		if grandparent == nil {
			deplClient := deplFactory.New(fallback.Addr + ":" + strconv.Itoa(deployer.Port))
			status := deplClient.Fallback(deploymentId, myself, location.ID())
			if status != http.StatusOK {
				log.Debugf("tried to fallback to %s, got %d", fallback.Id, status)
				hTable.removeDeployment(deploymentId)
			}
			continue
		}

		newParentChan := hTable.setDeploymentAsOrphan(deploymentId)

		var (
			locations []s2.CellID
			status    int
		)
		locations, status = archimedesClient.GetClientCentroids(deploymentId)
		if status == http.StatusNotFound {
			autoClient := autoFactory.New(servers.AutonomicLocalHostPort)
			var myLoc s2.CellID
			myLoc, status = autoClient.GetLocation()
			if status != http.StatusOK {
				log.Warnf("could not get centroids location, nor my location (%d)", status)
			}
			locations = append(locations, myLoc)
		} else if status != http.StatusOK {
			log.Errorf("got status %d while trying to get centroids for deployment %s", status, deploymentId)
		}

		deplClient := deplFactory.New(grandparent.Addr + ":" + strconv.Itoa(deployer.Port))
		status = deplClient.WarnOfDeadChild(deploymentId, deadParent.Id, myself, alternatives, locations)
		if status != http.StatusOK {
			log.Errorf("got status %d while renegotiating parent %s with %s for deployment %s", status,
				deadParent, grandparent.Id, deploymentId)
			continue
		}

		go waitForNewDeploymentParent(deploymentId, newParentChan)
	}
}

func waitForNewDeploymentParent(deploymentId string, newParentChan <-chan string) {
	waitingTimer := time.NewTimer(waitForNewParentTimeout * time.Second)

	log.Debugf("waiting new parent for %s", deploymentId)

	select {
	case <-waitingTimer.C:
		log.Debugf("falling back to %s", fallback)
		deplClient := deplFactory.New(fallback.Addr + ":" + strconv.Itoa(deployer.Port))
		status := deplClient.Fallback(deploymentId, myself, location.ID())
		if status != http.StatusOK {
			log.Debugf("tried to fallback to %s, got %d", fallback, status)
			return
		}
		return
	case newParentId := <-newParentChan:
		log.Debugf("got new parent %s for deployment %s", newParentId, deploymentId)
		return
	}
}

func attemptToExtend(deploymentId string, targetNode *utils.Node, config *api.ExtendDeploymentConfig, maxHops int,
	alternatives map[string]*utils.Node, exploringTTL int) {
	var extendTimer *time.Timer

	toExclude := map[string]interface{}{}
	toExclude[myself.Id] = nil

	for _, child := range config.Children {
		toExclude[child.Id] = nil
	}

	suspectedChild.Range(func(key, value interface{}) bool {
		suspectedId := key.(typeSuspectedChildMapKey)
		toExclude[suspectedId] = nil
		return true
	})

	targetId := ""
	if targetNode != nil {
		targetId = targetNode.Id
	}

	log.Debugf("attempting to extend %s to %s excluding %+v", deploymentId, targetId, toExclude)

	var (
		success        bool
		tries          = 0
		nodeToExtendTo = targetNode
	)
	for {
		if targetNode == nil {
			targetNode = getAlternative(alternatives, config.Locations, maxHops, toExclude)
		}

		if targetNode != nil {
			nodeToExtendTo = targetNode
			success = extendDeployment(deploymentId, nodeToExtendTo, config.Children, exploringTTL)
			if success {
				break
			}

			toExclude[nodeToExtendTo.Id] = nil
		}

		if tries == 5 {
			log.Debugf("failed to extend deployment %s", deploymentId)
			targetNode = myself
			for _, child := range config.Children {
				extendDeployment(deploymentId, child, nil, exploringTTL)
			}
			return
		}

		tries++
		extendTimer = time.NewTimer(extendAttemptTimeout * time.Second)
		<-extendTimer.C
	}

	if len(config.Locations) > 0 {
		archClient := archFactory.New(nodeToExtendTo.Addr + ":" + strconv.Itoa(archimedes.Port))
		archClient.SetExploringCells(deploymentId, config.Locations)
	}

	if len(toExclude) > 0 {
		autoClient := autoFactory.New(nodeToExtendTo.Addr + ":" + strconv.Itoa(autonomic.Port))
		toExcludeArr := make([]string, len(toExclude))
		i := 0
		for node := range toExclude {
			toExcludeArr[i] = node
			i++
		}
		autoClient.BlacklistNodes(deploymentId, myself.Id, toExcludeArr...)
	}
}

func extendDeployment(deploymentId string, nodeToExtendTo *utils.Node, children []*utils.Node, exploringTTL int) bool {
	dto, ok := hTable.deploymentToDTO(deploymentId)
	if !ok {
		log.Errorf("hierarchy table does not contain deployment %s", deploymentId)
		return false
	}

	dto.Grandparent = hTable.getParent(deploymentId)
	dto.Parent = myself
	dto.Children = children
	depClient := deplFactory.New(nodeToExtendTo.Addr + ":" + strconv.Itoa(deployer.Port))

	log.Debugf("extending deployment %s to %s", deploymentId, nodeToExtendTo.Id)
	status := depClient.RegisterDeployment(deploymentId, dto.Static, dto.DeploymentYAMLBytes, dto.Grandparent,
		dto.Parent, dto.Children, exploringTTL)
	switch status {
	case http.StatusConflict:
		log.Debugf("deployment %s already exists at %s", deploymentId, nodeToExtendTo.Id)
		return false
	case http.StatusOK:
		log.Debugf("extended %s to %s sucessfully", deploymentId, nodeToExtendTo.Id)
		hTable.addChild(deploymentId, nodeToExtendTo, exploringTTL != api.NotExploringTTL)
		fallthrough
	case http.StatusNoContent:
		log.Debugf("%s took deployment %s children %+v", nodeToExtendTo.Id, deploymentId, children)
		suspectedDeployments.Delete(deploymentId)
		return true
	default:
		log.Errorf("got %d while extending deployment %s to %s", status, deploymentId, nodeToExtendTo.Id)
		return false
	}
}

func loadFallbackHostname(filename string) *utils.Node {
	filePtr, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	var node utils.Node
	err = json.NewDecoder(filePtr).Decode(&node)
	if err != nil {
		panic(err)
	}

	return &node
}

func getAlternative(alternatives map[string]*utils.Node, targetLocations []s2.CellID, maxHops int,
	toExclude map[string]interface{}) (alternative *utils.Node) {
	for nodeId, node := range alternatives {
		delete(alternatives, nodeId)
		alternative = node
		return
	}

	var found bool
	alternative, found = getNodeCloserTo(targetLocations, maxHops, toExclude)
	if found {
		log.Debugf("trying %s", alternative)
	}

	return
}
