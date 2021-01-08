package deployer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	archimedesAPI "github.com/bruno-anjos/cloud-edge-deployment/api/archimedes"
	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/servers"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/archimedes"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/utils"

	"github.com/docker/go-connections/nat"
	"github.com/golang/geo/s2"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
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
	defer e.RUnlock()

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
		e.newParentChan <- e.parent.ID
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
	defer e.RUnlock()

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
	if _, ok := e.children.Load(child.ID); ok {
		return
	}

	e.children.Store(child.ID, &child)
	atomic.AddInt32(&e.numChildren, 1)
}

func (e *hierarchyEntry) removeChild(childID string) {
	e.children.Delete(childID)
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
	}

	atomic.StoreInt32(&e.orphan, notOrphanValue)

	return nil
}

func (e *hierarchyEntry) getNumChildren() int {
	return int(atomic.LoadInt32(&e.numChildren))
}

func (e *hierarchyEntry) getChildren() map[string]utils.Node {
	entryChildren := map[string]utils.Node{}

	e.children.Range(func(key, value interface{}) bool {
		childID := key.(typeChildMapKey)
		child := value.(typeChildMapValue)
		entryChildren[childID] = *child

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
		Mutex:            sync.Mutex{},
	}
}

func (t *hierarchyTable) updateDeployment(deploymentID string, parent, grandparent *utils.Node) bool {
	value, ok := t.hierarchyEntries.Load(deploymentID)
	if !ok {
		return false
	}

	entry := value.(typeHierarchyEntriesMapValue)

	parentID := nilString
	if parent != nil {
		parentID = parent.ID
		entry.setParent(*parent)
	} else {
		entry.removeParent()
	}

	grandparentID := nilString
	if grandparent != nil {
		grandparentID = grandparent.ID
		entry.setGrandparent(*grandparent)
	} else {
		entry.removeGrandparent()
	}

	log.Debugf("deployment %s updated with parent %s and grandparent %s", deploymentID, parentID, grandparentID)

	if parent != nil {
		autonomicClient.SetDeploymentParent(servers.AutonomicLocalHostPort, deploymentID, parent)
		deplClient := deplFactory.New()
		addr := parent.Addr + ":" + strconv.Itoa(deployer.Port)

		status := deplClient.PropagateLocationToHorizon(addr, deploymentID, myself, location.ID(), 0, api.Add)
		if status != http.StatusOK {
			log.Errorf("got status %d while trying to propagate location to %s for deployment %s", status,
				parent.ID, deploymentID)
		}
	}

	return true
}

func (t *hierarchyTable) addDeployment(dto *api.DeploymentDTO, depthFactor float64, exploringTTL int) bool {
	var static int32
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

	parentID := nilString
	if dto.Parent != nil {
		parentID = dto.Parent.ID
	}

	grandparentID := nilString
	if dto.Grandparent != nil {
		grandparentID = dto.Grandparent.ID
	}

	log.Debugf("deployment %s has parent %s and grandparent %s", dto.DeploymentID, parentID, grandparentID)

	_, loaded := t.hierarchyEntries.LoadOrStore(dto.DeploymentID, entry)
	if loaded {
		return false
	}

	autonomicClient.RegisterDeployment(servers.AutonomicLocalHostPort, dto.DeploymentID,
		autonomic.StrategyIdealLatencyID, depthFactor, exploringTTL)

	if dto.Parent != nil {
		autonomicClient.SetDeploymentParent(servers.AutonomicLocalHostPort, dto.DeploymentID, dto.Parent)
		deplClient := deplFactory.New()
		addr := dto.Parent.Addr + ":" + strconv.Itoa(deployer.Port)

		status := deplClient.PropagateLocationToHorizon(addr, dto.DeploymentID, myself, location.ID(), 0, api.Add)
		if status != http.StatusOK {
			log.Errorf("got status %d while trying to propagate location to %s for deployment %s", status,
				dto.Parent.ID, dto.DeploymentID)
		}
	}

	return true
}

func (t *hierarchyTable) removeDeployment(deploymentID string) {
	value, ok := t.hierarchyEntries.Load(deploymentID)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	if entry.getNumChildren() != 0 {
		log.Errorf("removing deployment that is still someones father")

		return
	}

	status := autonomicClient.DeleteDeployment(servers.AutonomicLocalHostPort, deploymentID)
	if status != http.StatusOK {
		log.Errorf("got status code %d from autonomic while deleting %s", status, deploymentID)

		return
	}

	var instances map[string]*archimedesAPI.Instance

	instances, status = archimedesClient.GetDeployment(servers.ArchimedesLocalHostPort, deploymentID)
	if status != http.StatusOK {
		log.Errorf("got status %d while requesting deployment %s instances", status, deploymentID)

		return
	}

	status = archimedesClient.DeleteDeployment(servers.ArchimedesLocalHostPort, deploymentID)
	if status != http.StatusOK {
		log.Errorf("got status code %d from archimedes", status)

		return
	}

	deploymentYAML := api.DeploymentYAML{}

	err := yaml.Unmarshal(t.getDeploymentConfig(deploymentID), &deploymentYAML)
	if err != nil {
		log.Panic(err)
	}

	for instanceID, instance := range instances {
		var (
			outport string
			ports   []nat.PortBinding
		)

		for _, ports = range instance.PortTranslation {
			outport = ports[0].HostPort
		}

		status = schedulerClient.StopInstance(servers.SchedulerLocalHostPort, instanceID,
			fmt.Sprintf("%s:%s", nodeIP, outport),
			deploymentYAML.RemovePath)
		if status != http.StatusOK {
			log.Errorf("got status code %d from scheduler", status)

			return
		}

		heartbeatsMap.Delete(instanceID)
	}

	parent := t.getParent(deploymentID)
	if parent != nil {
		deplClient := deplFactory.New()
		addr := parent.Addr + ":" + strconv.Itoa(deployer.Port)

		status = deplClient.PropagateLocationToHorizon(addr, deploymentID, myself, location.ID(), 0, api.Remove)
		if status != http.StatusOK {
			log.Errorf("got status %d while propagating location to %s for deployment %s", status, parent.ID,
				deploymentID)
		}
	}

	t.hierarchyEntries.Delete(deploymentID)
}

func (t *hierarchyTable) hasDeployment(deploymentID string) bool {
	_, ok := t.hierarchyEntries.Load(deploymentID)

	return ok
}

func (t *hierarchyTable) setDeploymentParent(deploymentID string, parent *utils.Node) {
	value, ok := t.hierarchyEntries.Load(deploymentID)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)

	if parent == nil {
		entry.removeParent()
	} else {
		entry.setParent(*parent)
	}

	auxChildren := t.getChildren(deploymentID)
	if len(auxChildren) > 0 {
		deplClient := deplFactory.New()

		for _, child := range auxChildren {
			addr := child.Addr + ":" + strconv.Itoa(deployer.Port)
			deplClient.SetGrandparent(addr, deploymentID, parent)
		}
	}

	entry.setOrphan(false)
}

func (t *hierarchyTable) setDeploymentGrandparent(deploymentID string, grandparent *utils.Node) {
	value, ok := t.hierarchyEntries.Load(deploymentID)
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

func (t *hierarchyTable) setDeploymentAsOrphan(deploymentID string) <-chan string {
	value, ok := t.hierarchyEntries.Load(deploymentID)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)

	return entry.setOrphan(true)
}

func (t *hierarchyTable) addChild(deploymentID string, child *utils.Node, exploring bool) {
	value, ok := t.hierarchyEntries.Load(deploymentID)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.addChild(*child)
	children.LoadOrStore(child.ID, child)
	autonomicClient.AddDeploymentChild(servers.AutonomicLocalHostPort, deploymentID, child)
	archimedesClient.AddDeploymentNode(servers.ArchimedesLocalHostPort, deploymentID, child, nodeLocCache.get(child),
		exploring)
}

func (t *hierarchyTable) removeChild(deploymentID, childID string) {
	value, ok := t.hierarchyEntries.Load(deploymentID)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.removeChild(childID)
	autonomicClient.RemoveDeploymentChild(servers.AutonomicLocalHostPort, deploymentID, childID)
	archimedesClient.DeleteDeploymentNode(servers.ArchimedesLocalHostPort, deploymentID, childID)
}

func (t *hierarchyTable) getChildren(deploymentID string) (children map[string]utils.Node) {
	value, ok := t.hierarchyEntries.Load(deploymentID)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)

	return entry.getChildren()
}

func (t *hierarchyTable) getParent(deploymentID string) *utils.Node {
	value, ok := t.hierarchyEntries.Load(deploymentID)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)
	if entry.hasParent() {
		parent := entry.getParent()

		return &parent
	}

	return nil
}

func (t *hierarchyTable) deploymentToDTO(deploymentID string) (*api.DeploymentDTO, bool) {
	value, ok := t.hierarchyEntries.Load(deploymentID)
	if !ok {
		return nil, false
	}

	entry := value.(typeHierarchyEntriesMapValue)

	return &api.DeploymentDTO{
		Children:            nil,
		Parent:              nil,
		Grandparent:         nil,
		DeploymentID:        deploymentID,
		Static:              entry.isStatic(),
		DeploymentYAMLBytes: entry.getDeploymentYAMLBytes(),
	}, true
}

func (t *hierarchyTable) isStatic(deploymentID string) bool {
	value, ok := t.hierarchyEntries.Load(deploymentID)
	if !ok {
		return false
	}

	entry := value.(typeHierarchyEntriesMapValue)

	return entry.isStatic()
}

func (t *hierarchyTable) removeParent(deploymentID string) bool {
	value, ok := t.hierarchyEntries.Load(deploymentID)
	if !ok {
		return false
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.removeParent()

	return true
}

func (t *hierarchyTable) getGrandparent(deploymentID string) *utils.Node {
	value, ok := t.hierarchyEntries.Load(deploymentID)
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

func (t *hierarchyTable) removeGrandparent(deploymentID string) {
	value, ok := t.hierarchyEntries.Load(deploymentID)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.removeGrandparent()
}

func (t *hierarchyTable) getDeployments() []string {
	var deploymentIds []string

	t.hierarchyEntries.Range(func(key, value interface{}) bool {
		deploymentID := key.(typeHierarchyEntriesMapKey)
		deploymentIds = append(deploymentIds, deploymentID)

		return true
	})

	return deploymentIds
}

func (t *hierarchyTable) getDeploymentConfig(deploymentID string) []byte {
	value, ok := t.hierarchyEntries.Load(deploymentID)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)

	return entry.getDeploymentYAMLBytes()
}

func (t *hierarchyTable) getDeploymentsWithParent(parentID string) (deploymentIds []string) {
	t.hierarchyEntries.Range(func(key, value interface{}) bool {
		deploymentID := key.(typeHierarchyEntriesMapKey)
		d := value.(typeHierarchyEntriesMapValue)

		if d.hasParent() && d.getParent().ID == parentID {
			deploymentIds = append(deploymentIds, deploymentID)
		}

		return true
	})

	return
}

func (t *hierarchyTable) toDTO() map[string]*api.HierarchyEntryDTO {
	entries := map[string]*api.HierarchyEntryDTO{}

	t.hierarchyEntries.Range(func(key, value interface{}) bool {
		deploymentID := key.(typeHierarchyEntriesMapKey)
		entry := value.(typeHierarchyEntriesMapValue)
		entries[deploymentID] = entry.toDTO()

		return true
	})

	return entries
}

const (
	alive     = 1
	suspected = 0
)

const (
	maxTries = 5
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

	log.Debugf("adding parent %s", parent.ID)

	t.parentEntries.Store(parent.ID, parentEntry)
}

func (t *parentsTable) hasParent(parentID string) bool {
	_, ok := t.parentEntries.Load(parentID)

	return ok
}

func (t *parentsTable) decreaseParentCount(parentID string) {
	value, ok := t.parentEntries.Load(parentID)
	if !ok {
		return
	}

	parentEntry := value.(typeParentEntriesMapValue)

	isZero := atomic.CompareAndSwapInt32(&parentEntry.NumOfDeployments, 1, 0)
	if isZero {
		t.removeParent(parentID)
	} else {
		atomic.AddInt32(&parentEntry.NumOfDeployments, -1)
	}
}

func (t *parentsTable) removeParent(parentID string) {
	log.Debugf("removing parent %s", parentID)
	t.parentEntries.Delete(parentID)
}

func (t *parentsTable) setParentUp(parentID string) {
	value, ok := t.parentEntries.Load(parentID)
	if !ok {
		return
	}

	atomic.StoreInt32(&value.(typeParentEntriesMapValue).IsUp, alive)
}

func (t *parentsTable) checkDeadParents() (deadParents []*utils.Node) {
	t.parentEntries.Range(func(key, value interface{}) bool {
		parentEntry := value.(typeParentEntriesMapValue)

		isAlive := atomic.CompareAndSwapInt32(&parentEntry.IsUp, alive, suspected)
		log.Debugf("setting parent %s as suspected", parentEntry.Parent.ID)
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
	deploymentIds := hTable.getDeploymentsWithParent(deadParent.ID)

	log.Debugf("renegotiating deployments %+v with parent %s", deploymentIds, deadParent.ID)

	for _, deploymentID := range deploymentIds {
		grandparent := hTable.getGrandparent(deploymentID)
		grandparentID := nilString

		if grandparent != nil {
			grandparentID = grandparent.ID
		}

		log.Debugf("my grandparent for %s is %s", deploymentID, grandparentID)

		if grandparent == nil {
			deplClient := deplFactory.New()
			addr := fallback.Addr + ":" + strconv.Itoa(deployer.Port)

			status := deplClient.Fallback(addr, deploymentID, myself, location.ID())
			if status != http.StatusOK {
				log.Debugf("tried to fallback to %s, got %d", fallback.ID, status)
				hTable.removeDeployment(deploymentID)
			}

			continue
		}

		newParentChan := hTable.setDeploymentAsOrphan(deploymentID)

		var (
			locations []s2.CellID
			status    int
		)

		locations, status = archimedesClient.GetClientCentroids(servers.ArchimedesLocalHostPort, deploymentID)
		if status == http.StatusNotFound {
			locations = append(locations, location.ID())
		} else if status != http.StatusOK {
			log.Errorf("got status %d while trying to get centroids for deployment %s", status, deploymentID)
		}

		deplClient := deplFactory.New()
		addr := grandparent.Addr + ":" + strconv.Itoa(deployer.Port)

		status = deplClient.WarnOfDeadChild(addr, deploymentID, deadParent.ID, myself, alternatives, locations)
		if status != http.StatusOK {
			log.Errorf("got status %d while renegotiating parent %s with %s for deployment %s", status,
				deadParent, grandparent.ID, deploymentID)

			continue
		}

		go waitForNewDeploymentParent(deploymentID, newParentChan)
	}
}

func waitForNewDeploymentParent(deploymentID string, newParentChan <-chan string) {
	waitingTimer := time.NewTimer(waitForNewParentTimeout * time.Second)

	log.Debugf("waiting new parent for %s", deploymentID)

	select {
	case <-waitingTimer.C:
		log.Debugf("falling back to %s", fallback)
		deplClient := deplFactory.New()
		addr := fallback.Addr + ":" + strconv.Itoa(deployer.Port)

		status := deplClient.Fallback(addr, deploymentID, myself, location.ID())
		if status != http.StatusOK {
			log.Debugf("tried to fallback to %s, got %d", fallback, status)

			return
		}

		return
	case newParentID := <-newParentChan:
		log.Debugf("got new parent %s for deployment %s", newParentID, deploymentID)

		return
	}
}

func attemptToExtend(deploymentID string, targetNode *utils.Node, config *api.ExtendDeploymentConfig, maxHops int,
	alternatives map[string]*utils.Node, exploringTTL int) {
	var extendTimer *time.Timer

	toExclude := map[string]interface{}{}
	toExclude[myself.ID] = nil

	for _, child := range config.Children {
		toExclude[child.ID] = nil
	}

	suspectedChild.Range(func(key, value interface{}) bool {
		suspectedID := key.(typeSuspectedChildMapKey)
		toExclude[suspectedID] = nil

		return true
	})

	targetID := ""
	if targetNode != nil {
		targetID = targetNode.ID
	}

	log.Debugf("attempting to extend %s to %s excluding %+v", deploymentID, targetID, toExclude)

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

			success = extendDeployment(deploymentID, nodeToExtendTo, config.Children, exploringTTL)
			if success {
				break
			}

			toExclude[nodeToExtendTo.ID] = nil
		}

		if tries == maxTries {
			log.Debugf("failed to extend deployment %s", deploymentID)

			for _, child := range config.Children {
				extendDeployment(deploymentID, child, nil, exploringTTL)
			}

			return
		}

		tries++

		extendTimer = time.NewTimer(extendAttemptTimeout * time.Second)
		<-extendTimer.C
	}

	if len(config.Locations) > 0 {
		addr := nodeToExtendTo.Addr + ":" + strconv.Itoa(archimedes.Port)
		archClient := archFactory.New(addr)

		archClient.SetExploringCells(addr, deploymentID, config.Locations)
	}

	if len(toExclude) > 0 {
		autoClient := autoFactory.New()
		addr := nodeToExtendTo.Addr + ":" + strconv.Itoa(autonomic.Port)
		toExcludeArr := make([]string, len(toExclude))
		i := 0

		for node := range toExclude {
			toExcludeArr[i] = node
			i++
		}

		autoClient.BlacklistNodes(addr, deploymentID, myself.ID, toExcludeArr, map[string]struct{}{myself.ID: {}})
	}
}

func extendDeployment(deploymentID string, nodeToExtendTo *utils.Node, children []*utils.Node, exploringTTL int) bool {
	dto, ok := hTable.deploymentToDTO(deploymentID)
	if !ok {
		log.Errorf("hierarchy table does not contain deployment %s", deploymentID)

		return false
	}

	dto.Grandparent = hTable.getParent(deploymentID)
	dto.Parent = myself
	dto.Children = children
	depClient := deplFactory.New()
	addr := nodeToExtendTo.Addr + ":" + strconv.Itoa(deployer.Port)

	log.Debugf("extending deployment %s to %s", deploymentID, nodeToExtendTo.ID)

	status := depClient.RegisterDeployment(addr, deploymentID, dto.Static, dto.DeploymentYAMLBytes, dto.Grandparent,
		dto.Parent, dto.Children, exploringTTL)
	switch status {
	case http.StatusConflict:
		log.Debugf("deployment %s already exists at %s", deploymentID, nodeToExtendTo.ID)

		return false
	case http.StatusOK:
		log.Debugf("extended %s to %s sucessfully", deploymentID, nodeToExtendTo.ID)
		hTable.addChild(deploymentID, nodeToExtendTo, exploringTTL != api.NotExploringTTL)

		fallthrough
	case http.StatusNoContent:
		log.Debugf("%s took deployment %s children %+v", nodeToExtendTo.ID, deploymentID, children)
		suspectedDeployments.Delete(deploymentID)

		return true
	default:
		log.Errorf("got %d while extending deployment %s to %s", status, deploymentID, nodeToExtendTo.ID)

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
	for nodeID, node := range alternatives {
		delete(alternatives, nodeID)

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
