package deployer

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	api "github.com/bruno-anjos/cloud-edge-deployment/api/deployer"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/strategies"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	log "github.com/sirupsen/logrus"
)

type (
	hierarchyEntry struct {
		DeploymentYAMLBytes []byte
		Parent              *utils.Node
		Grandparent         *utils.Node
		Children            sync.Map
		NumChildren         int32
		Static              bool
		IsOrphan            bool
		NewParentChan       chan<- string
		LinkOnly            bool
	}
)

func (e *hierarchyEntry) getChildren() map[string]*utils.Node {
	entryChildren := map[string]*utils.Node{}

	e.Children.Range(func(key, value interface{}) bool {
		childId := key.(typeChildMapKey)
		child := value.(typeChildMapValue)
		entryChildren[childId] = child
		return true
	})

	return entryChildren
}

func (e *hierarchyEntry) toDTO() *api.HierarchyEntryDTO {
	return &api.HierarchyEntryDTO{
		Parent:      e.Parent,
		Grandparent: e.Grandparent,
		Children:    e.getChildren(),
		Static:      e.Static,
		IsOrphan:    e.IsOrphan,
	}
}

type (
	typeChildMapKey   = string
	typeChildMapValue = *utils.Node

	hierarchyTable struct {
		hierarchyEntries sync.Map
		autonomicClient  *autonomic.Client
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

func (t *hierarchyTable) addDeployment(dto *api.DeploymentDTO) bool {
	entry := &hierarchyEntry{
		DeploymentYAMLBytes: dto.DeploymentYAMLBytes,
		Parent:              dto.Parent,
		Grandparent:         dto.Grandparent,
		Children:            sync.Map{},
		Static:              dto.Static,
		IsOrphan:            false,
		NewParentChan:       nil,
		LinkOnly:            true,
	}

	_, loaded := t.hierarchyEntries.LoadOrStore(dto.DeploymentId, entry)
	if loaded {
		return false
	}

	t.autonomicClient.RegisterService(dto.DeploymentId, strategies.StrategyIdealLatencyId)
	if dto.Parent != nil {
		log.Debugf("will set my parent as %s", dto.Parent.Addr)
		t.autonomicClient.SetServiceParent(dto.DeploymentId, dto.Parent.Addr)
	}
	return true
}

func (t *hierarchyTable) removeDeployment(deploymentId string) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	if entry.NumChildren > 0 {
		log.Debugf("setting deployment %s as linkonly", deploymentId)
		entry.LinkOnly = true
		deploymentChildren := entry.getChildren()
		for childId := range deploymentChildren {
			log.Debugf("redirecting %s linkonly to %s", deploymentId, childId)
			archimedesClient.Redirect(deploymentId, childId, -1)
			break
		}
	} else {
		archimedesClient.DeleteService(deploymentId)
		t.hierarchyEntries.Delete(deploymentId)
		t.autonomicClient.DeleteService(deploymentId)
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
	entry.Parent = parent

	if entry.NewParentChan != nil {
		entry.NewParentChan <- parent.Id
		close(entry.NewParentChan)
		entry.NewParentChan = nil
	}

	entry.IsOrphan = false
}

func (t *hierarchyTable) setDeploymentGrandparent(deploymentId string, grandparent *utils.Node) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.Grandparent = grandparent
}

func (t *hierarchyTable) setDeploymentAsOrphan(deploymentId string) <-chan string {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.IsOrphan = true
	newParentChan := make(chan string)
	entry.NewParentChan = newParentChan

	return newParentChan
}

func (t *hierarchyTable) addChild(deploymentId string, child *utils.Node) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.Children.Store(child.Id, child)
	atomic.AddInt32(&entry.NumChildren, 1)

	t.autonomicClient.AddServiceChild(deploymentId, child.Id)

	return
}

func (t *hierarchyTable) removeChild(deploymentId, childId string) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.Children.Delete(childId)
	t.autonomicClient.RemoveServiceChild(deploymentId, childId)

	isZero := atomic.CompareAndSwapInt32(&entry.NumChildren, 1, 0)
	if isZero {
		if entry.LinkOnly {
			t.removeDeployment(deploymentId)
		}
	} else {
		atomic.AddInt32(&entry.NumChildren, -1)
	}

	return
}

func (t *hierarchyTable) getChildren(deploymentId string) (children map[string]*utils.Node) {
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

	return entry.Parent
}

func (t *hierarchyTable) deploymentToDTO(deploymentId string) (*api.DeploymentDTO, bool) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil, false
	}

	entry := value.(typeHierarchyEntriesMapValue)

	return &api.DeploymentDTO{
		Parent:              entry.Parent,
		Grandparent:         entry.Grandparent,
		DeploymentId:        deploymentId,
		Static:              entry.Static,
		DeploymentYAMLBytes: entry.DeploymentYAMLBytes,
	}, true
}

func (t *hierarchyTable) isStatic(deploymentId string) bool {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return false
	}

	entry := value.(typeHierarchyEntriesMapValue)
	return entry.Static
}

func (t *hierarchyTable) removeParent(deploymentId string) bool {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return false
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.Parent = nil

	return true
}

func (t *hierarchyTable) getGrandparent(deploymentId string) *utils.Node {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)

	return entry.Grandparent
}

func (t *hierarchyTable) removeGrandparent(deploymentId string) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.Grandparent = nil
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
	return entry.DeploymentYAMLBytes
}

func (t *hierarchyTable) getDeploymentsWithParent(parentId string) (deploymentIds []string) {
	t.hierarchyEntries.Range(func(key, value interface{}) bool {
		deploymentId := key.(typeHierarchyEntriesMapKey)
		deployment := value.(typeHierarchyEntriesMapValue)

		if deployment.Parent.Id == parentId {
			deploymentIds = append(deploymentIds, deploymentId)
		}

		return true
	})

	return
}

func (t *hierarchyTable) setLinkOnly(deploymentId string, linkOnly bool) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.LinkOnly = linkOnly
}

func (t *hierarchyTable) isLinkOnly(deploymentId string) (linkOnly bool) {
	linkOnly = false

	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	ok = true

	entry := value.(typeHierarchyEntriesMapValue)
	linkOnly = entry.LinkOnly
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
			panic("TODO fallback")
		}

		newParentChan := hTable.setDeploymentAsOrphan(deploymentId)

		deplClient := deployer.NewDeployerClient(grandparent.Addr + ":" + strconv.Itoa(deployer.Port))
		status := deplClient.WarnOfDeadChild(deploymentId, deadParent.Id, myself, alternatives)
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
		panic("TODO fallback")
	case newParentId := <-newParentChan:
		log.Debugf("got new parent %s for deployment %s", newParentId, deploymentId)
		return
	}
}

func attemptToExtend(deploymentId string, target string, grandchild *utils.Node, maxHops int,
	alternatives map[string]*utils.Node) {
	var extendTimer *time.Timer

	toExclude := map[string]struct{}{myself.Id: {}}
	if grandchild != nil {
		toExclude[grandchild.Id] = struct{}{}
	}
	suspectedChild.Range(func(key, value interface{}) bool {
		suspectedId := key.(typeSuspectedChildMapKey)
		toExclude[suspectedId] = struct{}{}
		return true
	})

	log.Debugf("attempting to extend %s to %s excluding %+v", deploymentId, target, toExclude)

	var (
		success      bool
		newChildAddr = target
		tries        = 0
	)
	for !success {
		if newChildAddr == "" {
			if len(alternatives) > 0 {
				for alternative := range alternatives {
					newChildAddr = alternative
					break
				}
				delete(alternatives, newChildAddr)
			} else {
				newChildAddr = getNodeCloserTo(location, maxHops, toExclude)
				log.Debugf("trying %s", newChildAddr)
			}
			toExclude[newChildAddr] = struct{}{}
		}

		if newChildAddr != "" {
			success = extendDeployment(deploymentId, newChildAddr, grandchild)
		}

		if tries == 5 {
			log.Errorf("failed to extend deployment %s", deploymentId)
			newChildAddr = myself.Id
			extendDeployment(deploymentId, newChildAddr, grandchild)
			return
		}

		tries++
		extendTimer = time.NewTimer(extendAttemptTimeout * time.Second)
		<-extendTimer.C
	}
}

func extendDeployment(deploymentId, childAddr string, grandChild *utils.Node) bool {
	dto, ok := hTable.deploymentToDTO(deploymentId)
	if !ok {
		log.Errorf("hierarchy table does not contain deployment %s", deploymentId)
		return false
	}

	childGrandparent := hTable.getParent(deploymentId)
	if childGrandparent != nil && childGrandparent.Id == myself.Id {
		dto.Grandparent = nil
	} else {
		dto.Grandparent = childGrandparent
	}
	dto.Parent = myself

	deployerHostPort := addPortToAddr(childAddr)

	childId := childAddr

	if grandChild != nil && grandChild.Id == childId {
		return false
	}

	child := utils.NewNode(childId, childAddr)

	depClient := deployer.NewDeployerClient(deployerHostPort)
	status := depClient.AskCanTakeChild(deploymentId, childId)
	if status == http.StatusConflict {
		return false
	} else if status != http.StatusOK {
		log.Debugf("got status code %d asking if it could take child %s", status, childId)
		return false
	}

	log.Debugf("extending deployment %s to %s", deploymentId, childId)
	status = depClient.RegisterService(deploymentId, dto.Static, dto.DeploymentYAMLBytes, dto.Parent, dto.Grandparent)
	if status == http.StatusConflict {
		log.Debugf("deployment %s is already present in %s", deploymentId, childId)
	} else if status != http.StatusOK {
		log.Errorf("got %d while extending deployment %s to %s", status, deploymentId, childAddr)
		return false
	}

	if grandChild != nil {
		log.Debugf("telling %s to take grandchild %s for deployment %s", childId, grandChild.Id, deploymentId)
		status = depClient.WarnToTakeChild(deploymentId, grandChild)
		if status != http.StatusOK {
			log.Errorf("got status %d while attempting to tell %s to take %s as child", status, childId,
				grandChild.Id)
			return false
		}
	}

	log.Debugf("extended %s to %s sucessfully", deploymentId, childId)
	if child.Id != myself.Id {
		hTable.addChild(deploymentId, child)
		children.Store(childId, child)
	}

	return true
}

func resetToFallback() {

}

func loadFallbackHostname(filename string) string {
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	return string(fileBytes)
}
