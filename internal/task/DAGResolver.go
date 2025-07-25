package task

import (
	"maps"
	"slices"
)

type Node interface {
	GetUid() string
	GetNext() []string
	AddNextUid(string)
	SetNextUids([]string)
}

func MakeDAGResolver(nodes []Node) *DAGResolver {
	resolver := DAGResolver{
		nodeMap:  make(map[string]Node),
		revIndex: make(map[string][]Node),
	}
	for _, node := range nodes {
		resolver.AddNode(node)
	}
	resolver.RefreshTables()

	return &resolver
}

type DAGResolver struct {
	nodeMap  map[string]Node   //node.Uid -> Node
	revIndex map[string][]Node //node.Uid -> incoming Node
}

func (d *DAGResolver) AddNode(n Node) { d.nodeMap[n.GetUid()] = n }

func (d *DAGResolver) GetNode(uid string) (Node, bool) {
	val, ok := d.nodeMap[uid]
	return val, ok
}

func (d *DAGResolver) NodeExists(uid string) bool {
	_, ok := d.nodeMap[uid]
	return ok
}

func (d *DAGResolver) GetNextUid(uid string) []string {
	node, ok := d.GetNode(uid)

	if !ok {
		return []string{}
	} else {
		return node.GetNext()
	}
}

func (d *DAGResolver) rebuildRevIndex() {
	revIndex := make(map[string][]Node)

	for _, node := range d.nodeMap {
		nextUids := node.GetNext()
		for _, nextUid := range nextUids {
			if _, ok := revIndex[nextUid]; !ok {
				revIndex[nextUid] = []Node{}
			}
			revIndex[nextUid] = append(revIndex[nextUid], node)
		}
	}

	d.revIndex = revIndex
}

func (d *DAGResolver) RefreshTables() {
	d.rebuildRevIndex()
}

func (d *DAGResolver) GetNodeMap() map[string]Node { return d.nodeMap }

func (d *DAGResolver) GetIncomingNodes(uid string) ([]Node, bool) {
	nodes, ok := d.revIndex[uid]

	return nodes, ok
}

func (d *DAGResolver) GetConvergencePoints() map[string]int {
	incomingCounts := d.CountIncomingEdges(nil)
	filterFunc := func(k string, v int) bool { return v > 1 }
	return FilterMap(incomingCounts, filterFunc)
}

func (d *DAGResolver) GetDivergencePoints() []Node {
	nodes := []Node{}
	for _, v := range d.GetNodeMap() {
		if len(v.GetNext()) > 1 {
			nodes = append(nodes, v)
		}
	}
	return nodes
}

func (d *DAGResolver) GetAllNodes(uids []string) []Node {
	nodes := []Node{}
	for _, uid := range uids {
		if node, ok := d.GetNode(uid); ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}
func (d *DAGResolver) GetDownstreamNodes(startNodes []Node) []Node {
	exploredNodes := make(map[string]struct{})
	downstreamNodes := []Node{}

	toExploreQueue := []Node{}
	for _, node := range startNodes {
		toExploreQueue = append(toExploreQueue, node)
	}

	for len(toExploreQueue) > 0 {
		node := toExploreQueue[0]
		downstreamNodes = append(downstreamNodes, node)
		toExploreQueue = toExploreQueue[1:]
		for _, nextUid := range node.GetNext() {
			nextNode, nextNodeExists := d.GetNode(nextUid)
			if _, ok := exploredNodes[nextUid]; !ok && nextNodeExists {
				toExploreQueue = append(toExploreQueue, nextNode)
				exploredNodes[nextUid] = struct{}{}
			}
		}
	}
	return downstreamNodes
}

func (d *DAGResolver) CountIncomingEdges(nodes []Node) map[string]int {
	counts := make(map[string]int)
	if len(nodes) == 0 || nodes == nil {
		allNodes := []Node{}
		for _, v := range d.GetNodeMap() {
			allNodes = append(allNodes, v)
		}
		nodes = allNodes
	}
	for _, node := range nodes {
		if nodes, hasIncoming := d.GetIncomingNodes(node.GetUid()); hasIncoming {
			counts[node.GetUid()] = len(nodes)
		} else {
			counts[node.GetUid()] = 0
		}
	}
	return counts
}

func (d *DAGResolver) GetLinearOrder() []Node {
	counts := d.CountIncomingEdges(nil)
	startUids := []string{}
	endUids := []string{}

	for k, v := range counts {
		if v == 0 {
			startUids = append(startUids, k)
		}
		if node, ok := d.GetNode(k); ok && len(node.GetNext()) == 0 {
			endUids = append(endUids, k)
		}
	}
	orderedNodes := d.customKahnsAlgorithm(startUids, endUids)

	return orderedNodes
}

func (d *DAGResolver) customKahnsAlgorithm(startUids []string, endUids []string) []Node {

	startNodes := d.GetAllNodes(startUids)
	allNodes := d.GetDownstreamNodes(startNodes)
	incomingEdgeCounts := d.CountIncomingEdges(allNodes)

	zeroDegreeFilter := func(k string, v int) bool { return v == 0 }
	zeroDegreeNodes := FilterMap(incomingEdgeCounts, zeroDegreeFilter)

	// Kahn's Algorithm: https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm
	inEdgesCounts := make(map[string]int)
	maps.Copy(inEdgesCounts, incomingEdgeCounts)
	orderedNodes := []Node{}
	for len(zeroDegreeNodes) > 0 {
		var uid string
		for k := range zeroDegreeNodes {
			uid = k
			break
		}

		delete(zeroDegreeNodes, uid)
		node, exists := d.GetNode(uid)
		if !exists {
			continue
		}
		orderedNodes = append(orderedNodes, node)
		if slices.Contains(endUids, node.GetUid()) {
			continue
		}
		for _, nextUid := range node.GetNext() {
			if _, exists := inEdgesCounts[nextUid]; exists {
				inEdgesCounts[nextUid]--
				if inEdgesCounts[nextUid] <= 0 {
					zeroDegreeNodes[nextUid] = 0
				}
			}
		}
	}
	return orderedNodes
}
