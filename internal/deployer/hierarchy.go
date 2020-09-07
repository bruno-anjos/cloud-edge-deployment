package deployer

import (
	"io/ioutil"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bruno-anjos/cloud-edge-deployment/internal/autonomic/strategies"
	"github.com/bruno-anjos/cloud-edge-deployment/internal/utils"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/autonomic"
	"github.com/bruno-anjos/cloud-edge-deployment/pkg/deployer"
	log "github.com/sirupsen/logrus"
)

type (
	HierarchyEntry struct {
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

func (e *HierarchyEntry) GetChildren() map[string]*utils.Node {
	entryChildren := map[string]*utils.Node{}

	e.Children.Range(func(key, value interface{}) bool {
		childId := key.(typeChildMapKey)
		child := value.(typeChildMapValue)
		entryChildren[childId] = child
		return true
	})

	return entryChildren
}

func (e *HierarchyEntry) ToDTO() *deployer.HierarchyEntryDTO {
	return &deployer.HierarchyEntryDTO{
		Parent:      e.Parent,
		Grandparent: e.Grandparent,
		Child:       e.GetChildren(),
		Static:      e.Static,
		IsOrphan:    e.IsOrphan,
	}
}

type (
	typeChildMapKey   = string
	typeChildMapValue = *utils.Node

	HierarchyTable struct {
		hierarchyEntries sync.Map
		autonomicClient  *autonomic.Client
	}

	typeHierarchyEntriesMapKey   = string
	typeHierarchyEntriesMapValue = *HierarchyEntry
)

func NewHierarchyTable() *HierarchyTable {
	return &HierarchyTable{
		hierarchyEntries: sync.Map{},
		autonomicClient:  autonomic.NewAutonomicClient(autonomic.AutonomicServiceName),
	}
}

func (t *HierarchyTable) AddDeployment(dto *deployer.DeploymentDTO) bool {
	entry := &HierarchyEntry{
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
	return true
}

func (t *HierarchyTable) RemoveDeployment(deploymentId string) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	if entry.NumChildren > 0 {
		log.Debugf("setting deployment %s as linkonly", deploymentId)
		entry.LinkOnly = true
		deploymentChildren := entry.GetChildren()
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

func (t *HierarchyTable) HasDeployment(deploymentId string) bool {
	_, ok := t.hierarchyEntries.Load(deploymentId)
	return ok
}

func (t *HierarchyTable) SetDeploymentParent(deploymentId string, parent *utils.Node) {
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

func (t *HierarchyTable) SetDeploymentAsOrphan(deploymentId string) <-chan string {
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

func (t *HierarchyTable) AddChild(deploymentId string, child *utils.Node) {
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

func (t *HierarchyTable) RemoveChild(deploymentId, childId string) {
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
			t.RemoveDeployment(deploymentId)
		}
	} else {
		atomic.AddInt32(&entry.NumChildren, -1)
	}

	return
}

func (t *HierarchyTable) GetChildren(deploymentId string) (children map[string]*utils.Node) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)
	return entry.GetChildren()
}

func (t *HierarchyTable) GetParent(deploymentId string) *utils.Node {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)

	return entry.Parent
}

func (t *HierarchyTable) DeploymentToDTO(deploymentId string) (*deployer.DeploymentDTO, bool) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil, false
	}

	entry := value.(typeHierarchyEntriesMapValue)

	return &deployer.DeploymentDTO{
		Parent:              entry.Parent,
		Grandparent:         entry.Grandparent,
		DeploymentId:        deploymentId,
		Static:              entry.Static,
		DeploymentYAMLBytes: entry.DeploymentYAMLBytes,
	}, true
}

func (t *HierarchyTable) IsStatic(deploymentId string) bool {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return false
	}

	entry := value.(typeHierarchyEntriesMapValue)
	return entry.Static
}

func (t *HierarchyTable) RemoveParent(deploymentId string) bool {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return false
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.Parent = nil

	return true
}

func (t *HierarchyTable) GetGrandparent(deploymentId string) *utils.Node {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)

	return entry.Grandparent
}

func (t *HierarchyTable) RemoveGrandparent(deploymentId string) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.Grandparent = nil
}

func (t *HierarchyTable) GetDeployments() []string {
	var deploymentIds []string

	t.hierarchyEntries.Range(func(key, value interface{}) bool {
		deploymentId := key.(typeHierarchyEntriesMapKey)
		deploymentIds = append(deploymentIds, deploymentId)
		return true
	})

	return deploymentIds
}

func (t *HierarchyTable) GetDeploymentConfig(deploymentId string) []byte {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return nil
	}

	entry := value.(typeHierarchyEntriesMapValue)
	return entry.DeploymentYAMLBytes
}

func (t *HierarchyTable) GetDeploymentsWithParent(parentId string) (deploymentIds []string) {
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

func (t *HierarchyTable) SetLinkOnly(deploymentId string, linkOnly bool) {
	value, ok := t.hierarchyEntries.Load(deploymentId)
	if !ok {
		return
	}

	entry := value.(typeHierarchyEntriesMapValue)
	entry.LinkOnly = linkOnly
}

func (t *HierarchyTable) IsLinkOnly(deploymentId string) (linkOnly bool) {
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

func (t *HierarchyTable) ToDTO() map[string]*deployer.HierarchyEntryDTO {
	entries := map[string]*deployer.HierarchyEntryDTO{}

	t.hierarchyEntries.Range(func(key, value interface{}) bool {
		deploymentId := key.(typeHierarchyEntriesMapKey)
		entry := value.(typeHierarchyEntriesMapValue)
		entries[deploymentId] = entry.ToDTO()
		return true
	})

	return entries
}

const (
	alive     = 1
	suspected = 0
)

type (
	ParentsEntry struct {
		Parent           *utils.Node
		NumOfDeployments int32
		IsUp             int32
	}

	ParentsTable struct {
		parentEntries sync.Map
	}

	typeParentEntriesMapValue = *ParentsEntry
)

func NewParentsTable() *ParentsTable {
	return &ParentsTable{
		parentEntries: sync.Map{},
	}
}

func (t *ParentsTable) AddParent(parent *utils.Node) {
	parentEntry := &ParentsEntry{
		Parent:           parent,
		NumOfDeployments: 1,
		IsUp:             alive,
	}

	t.parentEntries.Store(parent.Id, parentEntry)
}

func (t *ParentsTable) HasParent(parentId string) bool {
	_, ok := t.parentEntries.Load(parentId)
	return ok
}

func (t *ParentsTable) DecreaseParentCount(parentId string) {
	value, ok := t.parentEntries.Load(parentId)
	if !ok {
		return
	}

	parentEntry := value.(typeParentEntriesMapValue)
	isZero := atomic.CompareAndSwapInt32(&parentEntry.NumOfDeployments, 1, 0)
	if isZero {
		t.RemoveParent(parentId)
	} else {
		atomic.AddInt32(&parentEntry.NumOfDeployments, -1)
	}

	return
}

func (t *ParentsTable) RemoveParent(parentId string) {
	t.parentEntries.Delete(parentId)
}

func (t *ParentsTable) SetParentUp(parentId string) {
	value, ok := t.parentEntries.Load(parentId)
	if !ok {
		return
	}

	parentEntry := value.(typeParentEntriesMapValue)
	atomic.StoreInt32(&parentEntry.IsUp, alive)
}

func (t *ParentsTable) CheckDeadParents() (deadParents []*utils.Node) {
	t.parentEntries.Range(func(key, value interface{}) bool {
		parentEntry := value.(typeParentEntriesMapValue)

		isAlive := atomic.CompareAndSwapInt32(&parentEntry.IsUp, alive, suspected)
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

func renegotiateParent(deadParent *utils.Node) {
	deploymentIds := hierarchyTable.GetDeploymentsWithParent(deadParent.Id)

	log.Debugf("renegotiating deployments %+v with parent %s", deploymentIds, deadParent.Id)

	for _, deploymentId := range deploymentIds {
		grandparent := hierarchyTable.GetGrandparent(deploymentId)
		if grandparent == nil {
			panic("TODO fallback")
		}

		newParentChan := hierarchyTable.SetDeploymentAsOrphan(deploymentId)

		req := utils.BuildRequest(http.MethodPost, grandparent.Addr, GetDeadChildPath(deploymentId, deadParent.Id),
			myself)
		status, _ := utils.DoRequest(httpClient, req, nil)
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

func attemptToExtend(deploymentId string, grandchild *utils.Node, location string, maxHops int) {
	extendTimer := time.NewTimer(extendAttemptTimeout * time.Second)

	toExclude := map[string]struct{}{}
	if grandchild != nil {
		toExclude[grandchild.Id] = struct{}{}
	}
	suspectedChild.Range(func(key, value interface{}) bool {
		suspectedId := key.(typeSuspectedChildMapKey)
		toExclude[suspectedId] = struct{}{}
		return true
	})

	var (
		success      bool
		newChildAddr string
		tries        = 0
	)
	for !success {
		newChildAddr = getNodeCloserTo(location, maxHops, toExclude)
		success = extendDeployment(deploymentId, newChildAddr, grandchild)
		tries++
		if tries == 10 {
			log.Errorf("failed to extend deployment %s", deploymentId)
			return
		}
		<-extendTimer.C
	}
}

func extendDeployment(deploymentId, childAddr string, grandChild *utils.Node) bool {
	dto, ok := hierarchyTable.DeploymentToDTO(deploymentId)
	if !ok {
		log.Errorf("hierarchy table does not contain deployment %s", deploymentId)
		return false
	}

	childGrandparent := hierarchyTable.GetParent(deploymentId)
	dto.Grandparent = childGrandparent
	dto.Parent = myself

	deployerHostPort := addPortToAddr(childAddr)

	childId, err := getDeployerIdFromAddr(childAddr)
	if err != nil {
		return false
	}

	if grandChild != nil && grandChild.Id == childId {
		return false
	}

	child := utils.NewNode(childId, childAddr)

	log.Debugf("extending deployment %s to %s", deploymentId, childId)

	req := utils.BuildRequest(http.MethodPost, deployerHostPort, GetDeploymentsPath(), dto)
	status, _ := utils.DoRequest(httpClient, req, nil)
	if status == http.StatusConflict {
		log.Debugf("deployment %s is already present in %s", deploymentId, childId)
	} else if status != http.StatusOK {
		log.Errorf("got %d while extending deployment %s to %s", status, deploymentId, childAddr)
		return false
	}

	if grandChild != nil {
		log.Debugf("telling %s to take grandchild %s for deployment %s", childId, grandChild.Id, deploymentId)
		req = utils.BuildRequest(http.MethodPost, deployerHostPort, GetDeploymentChildPath(deploymentId, grandChild.Id),
			grandChild)
		status, _ = utils.DoRequest(httpClient, req, nil)
		if status != http.StatusOK {
			log.Errorf("got status %d while attempting to tell %s to take %s as child", status, childId,
				grandChild.Id)
			return false
		}
	}

	log.Debugf("extended %s to %s sucessfully", deploymentId, childId)
	hierarchyTable.AddChild(deploymentId, child)
	children.Store(childId, child)

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
