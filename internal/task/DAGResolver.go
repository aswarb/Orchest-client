package task

import (
	"maps"
)

type Node interface {
	GetUid() string
	GetNext() []string
	AddNextUid(string)
	SetNextUids([]string)
}

type Segment interface {
	GetUid() string
	GetMemberUids() []string
	GetEndpointUids() []string
	AddMemberUid(string)
	SetMemberUids([]string)
}

type DAGResolver struct {
	nodeMap       map[string]Node                //node.Uid -> Node
	revIndex      map[string][]Node              //node.Uid -> incoming Node
	segmentMap    map[string]Segment             //segment.Uid -> Segment
	segmentRevMap map[string]map[string]struct{} //Node.Uid -> set(segment.Uid)
}

func (d *DAGResolver) AddNode(n Node)       { d.nodeMap[n.GetUid()] = n }
func (d *DAGResolver) AddSegment(s Segment) { d.segmentMap[s.GetUid()] = s }

func (d *DAGResolver) GetNode(uid string) (Node, bool) {
	val, ok := d.nodeMap[uid]
	return val, ok
}

func (d *DAGResolver) NodeExists(uid string) bool {
	_, ok := d.nodeMap[uid]
	return ok
}

func (d *DAGResolver) GetSegment(uid string) (Segment, bool) {
	val, ok := d.segmentMap[uid]
	return val, ok
}

func (d *DAGResolver) GetNextUid(uid string) []string {
	node, ok := d.GetNode(uid)

	if !ok {
		return []string{}
	} else {
		return node.GetNext()
	}
}

func (d *DAGResolver) GetNextBranchPoint() {}
func (d *DAGResolver) rebuildSegmentRevIndex() {
	segments := d.getSegmentMap()
	revIndex := make(map[string](map[string]struct{}))
	for _, segment := range segments {
		sUid := segment.GetUid()

		memberUids := segments[sUid].GetMemberUids()
		for _, nUid := range memberUids {
			_, exists := revIndex[nUid]
			if !exists {
				revIndex[nUid] = make(map[string]struct{})
			}
			revIndex[nUid][sUid] = struct{}{}
		}
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
	d.rebuildSegmentRevIndex()
	d.rebuildRevIndex()
}

func (d *DAGResolver) getNodeMap() map[string]Node       { return d.nodeMap }
func (d *DAGResolver) getSegmentMap() map[string]Segment { return d.segmentMap }

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
	for _, v := range d.getNodeMap() {
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
		exploredNodes[node.GetUid()] = struct{}{}
		toExploreQueue = toExploreQueue[:1]

		for _, nextUid := range node.GetNext() {
			nextNode, _ := d.GetNode(nextUid)
			if _, ok := exploredNodes[nextUid]; ok {
				toExploreQueue = append(toExploreQueue, nextNode)
			}

		}

	}

	return downstreamNodes
}

func (d *DAGResolver) CountIncomingEdges(nodes []Node) map[string]int {
	counts := make(map[string]int)

	if len(nodes) == 0 || nodes == nil {
		allNodes := []Node{}
		for _, v := range d.getNodeMap() {
			allNodes = append(allNodes, v)
		}
		nodes = allNodes
	}

	for _, node := range nodes {
		uid := node.GetUid()
		_, uidInMap := counts[uid]
		if uidInMap {
			counts[uid] = 0
		}
		for _, nextUid := range node.GetNext() {
			_, nextUidInMap := counts[nextUid]
			if nextUidInMap {
				counts[nextUid]++
			} else {
				counts[nextUid] = 1
			}
		}
	}
	return counts
}

func (d *DAGResolver) GetSegments(nUid string) (map[string]struct{}, bool) {
	segments, ok := d.segmentRevMap[nUid]

	return segments, ok
}

func (d *DAGResolver) GetLinearOrderFromSegment(sUid string) []Node {
	segment, _ := d.GetSegment(sUid)

	orderedNodes := d.customKahnsAlgorithm(segment.GetMemberUids(), segment.GetEndpointUids())

	return orderedNodes
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
		if exists {
			orderedNodes = append(orderedNodes, node)
		}
		for _, nextUid := range node.GetNext() {
			continueLoop := true
			for _, endUid := range endUids {
				if endUid == nextUid {
					continueLoop = false
					break
				}
			}
			if !continueLoop {
				break
			}
			_, exists := inEdgesCounts[nextUid]
			if exists {
				inEdgesCounts[nextUid]--
			}
			if inEdgesCounts[nextUid] <= 0 {
				zeroDegreeNodes[nextUid] = 0
			}
		}
	}
	return orderedNodes
}
