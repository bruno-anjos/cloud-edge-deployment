package deployer

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	archimedes2 "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
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
		deploymentYAMLBytes []byte
		parent              *utils.Node
		grandparent         *utils.Node
		children            sync.Map
		numChildren         int32
		static              int32
		orphan              int32
		newParentChan       chan<- string
		sync.RWMutex
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
		autonomicClient  *autonomic.Client
		sync.Mutex
	}

	typeHierarchyEntriesMapKey   = string
	typeHierarchyEntriesMapValue = *hierarchyEntry
)

func newHierarchyTable() *hierarchyTable {
	return &hierarchyTable{
		hierarchyEntries: sync.Map{},
		autonomicClient:  autonomic.NewAutonomicClient(autonomic.DefaultHostPort),
	}
}

func (t *hierarchyTable) addDeployment(dto *api.DeploymentDTO, exploring bool) bool {
	var (
		static int32
	)
	if dto.Static {
		static = staticValue
	} else {
		static = notStaticValue
	}

	entry := &hierarchyEntry{
		deploymentYAMLBytes: dto.DeploymentYAMLBytes,
		parent:              dto.Parent,
		children:            sync.Map{},
		static:              static,
		orphan:              notStaticValue,
		newParentChan:       nil,
	}

	_, loaded := t.hierarchyEntries.LoadOrStore(dto.DeploymentId, entry)
	if loaded {
		return false
	}

	t.autonomicClient.RegisterDeployment(dto.DeploymentId, autonomic.StrategyIdealLatencyId, exploring)
	if dto.Parent != nil {
		log.Debugf("will set my parent as %s", dto.Parent.Addr)
		t.autonomicClient.SetDeploymentParent(dto.DeploymentId, dto.Parent.Addr)
		deplClient := deployer.NewDeployerClient(dto.Parent.Id + ":" + strconv.Itoa(deployer.Port))
		status := deplClient.PropagateLocationToHorizon(dto.DeploymentId, myself.Id, location.ID(), 0)
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
		status := t.autonomicClient.DeleteDeployment(deploymentId)
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
	entry.setParent(*parent)

	auxChildren := t.getChildren(deploymentId)
	if len(auxChildren) > 0 {
		deplClient := deployer.NewDeployerClient("")
		for childId := range auxChildren {
			deplClient.SetHostPort(childId + ":" + strconv.Itoa(deployer.Port))
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
	entry.setGrandparent(*grandparent)
}

func (t *hierarchyTable) setDeploymentAsOrphan(deploymentId string) <-chan string {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)
	return entry.setOrphan(true)
}

func (t *hierarchyTable) addChild(deploymentId string, child *utils.Node) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.addChild(*child)
	children.LoadOrStore(child.Id, child)
	t.autonomicClient.AddDeploymentChild(deploymentId, child.Id)
	return
}

func (t *hierarchyTable) removeChild(deploymentId, childId string) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.removeChild(childId)
	t.autonomicClient.RemoveDeploymentChild(deploymentId, childId)
	removeTerminalLocsForChild(deploymentId, childId)

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
	var parent *utils.Node
	if entry.hasParent() {
		parentAux := entry.getParent()
		parent = &parentAux
	}

	return &api.DeploymentDTO{
		Parent:              parent,
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
		deployment := value.(typeHierarchyEntriesMapValue)

		if deployment.hasParent() && deployment.getParent().Id == parentId {
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
	t.parentEntries.Delete(parentId)
}

func (t *parentsTable) setParentUp(parentId string) {
	value, ok := t.parentEntries.Load(parentId)
	if !ok {
		return
	}

	parentEntry := value.(typeParentEntriesMapValue)
	atomic.StoreInt32(&parentEntry.IsUp, alive)
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
		if grandparent == nil {
			deplClient := deployer.NewDeployerClient(fallback + ":" + strconv.Itoa(deployer.Port))
			status := deplClient.Fallback(deploymentId, myself.Id, location.ID())
			if status != http.StatusOK {
				log.Debugf("tried to fallback to %s, got %d", fallback, status)
				hTable.removeDeployment(deploymentId)
			}
			continue
		}

		newParentChan := hTable.setDeploymentAsOrphan(deploymentId)

		locations, status := archimedesClient.GetClientCentroids(deploymentId)
		if status != http.StatusOK {
			log.Errorf("got status %d while trying to get centroids for deployment %s", status, deploymentId)
		}

		deplClient := deployer.NewDeployerClient(grandparent.Addr + ":" + strconv.Itoa(deployer.Port))
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
		deplClient := deployer.NewDeployerClient(fallback)
		status := deplClient.Fallback(deploymentId, myself.Id, location.ID())
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

func attemptToExtend(deploymentId, target string, config *api.ExtendDeploymentConfig, maxHops int,
	alternatives map[string]*utils.Node, isExploring bool) {
	var extendTimer *time.Timer

	toExclude := config.ToExclude
	toExclude[myself.Id] = nil
	for _, child := range config.Children {
		toExclude[child.Id] = nil
	}

	suspectedChild.Range(func(key, value interface{}) bool {
		suspectedId := key.(typeSuspectedChildMapKey)
		toExclude[suspectedId] = nil
		return true
	})

	log.Debugf("attempting to extend %s to %s excluding %+v", deploymentId, target, toExclude)

	var (
		success        bool
		tries          = 0
		nodeToExtendTo = utils.NewNode(target, target)
	)
	for !success {
		hasTarget := target != ""
		if !hasTarget {
			target = getAlternative(alternatives, config.Locations, maxHops, toExclude)
			hasTarget = target != ""
		}

		if hasTarget {
			nodeToExtendTo = utils.NewNode(target, target)
			success = extendDeployment(deploymentId, nodeToExtendTo, config.Children, isExploring)
			if success {
				break
			}
		}

		if tries == 5 {
			log.Debugf("failed to extend deployment %s", deploymentId)
			target = myself.Id
			for _, child := range config.Children {
				extendDeployment(deploymentId, child, nil, isExploring)
			}
			return
		}

		tries++
		extendTimer = time.NewTimer(extendAttemptTimeout * time.Second)
		<-extendTimer.C
	}

	if isExploring {
		id := deploymentId + "_" + nodeToExtendTo.Id
		log.Debugf("setting extension ")
		exploring.Store(id, &sync.Once{})
	}

	if len(config.Locations) > 0 {
		archClient := archimedes.NewArchimedesClient(nodeToExtendTo.Id + ":" + strconv.Itoa(archimedes.Port))
		archClient.SetExploringCells(deploymentId, config.Locations)
	}
}

func extendDeployment(deploymentId string, nodeToExtendTo *utils.Node, children []*utils.Node, exploring bool) bool {
	dto, ok := hTable.deploymentToDTO(deploymentId)
	if !ok {
		log.Errorf("hierarchy table does not contain deployment %s", deploymentId)
		return false
	}

	dto.Grandparent = hTable.getGrandparent(deploymentId)
	dto.Parent = myself
	dto.Children = children
	depClient := deployer.NewDeployerClient(nodeToExtendTo.Id + ":" + strconv.Itoa(deployer.Port))

	log.Debugf("extending deployment %s to %s", deploymentId, nodeToExtendTo.Id)
	status := depClient.RegisterDeployment(deploymentId, dto.Static, dto.DeploymentYAMLBytes, dto.Grandparent,
		dto.Parent, dto.Children, exploring)
	if status != http.StatusOK {
		log.Errorf("got %d while extending deployment %s to %s", status, deploymentId, nodeToExtendTo.Id)
		return false
	}

	hTable.addChild(deploymentId, nodeToExtendTo)
	log.Debugf("extended %s to %s sucessfully", deploymentId, nodeToExtendTo.Id)
	suspectedDeployments.Delete(deploymentId)

	return true
}

func loadFallbackHostname(filename string) string {
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	return string(fileBytes)
}

func getAlternative(alternatives map[string]*utils.Node, targetLocations []s2.CellID, maxHops int,
	toExclude map[string]interface{}) (result string) {
	if len(alternatives) > 0 {
		for alternative := range alternatives {
			result = alternative
			break
		}
		delete(alternatives, result)
		return
	}

	var found bool
	result, found = getNodeCloserTo(targetLocations, maxHops, toExclude)
	if found {
		log.Debugf("trying %s", result)
	}

	return
}
