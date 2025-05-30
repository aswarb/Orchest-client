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

func (d *DAGResolver) RefreshTables() {
	d.rebuildSegmentRevIndex()
}

func (d *DAGResolver) GetNodeLinearOrder() []Node {
	incomingEdgeCounts := d.CountIncomingEdges()
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

func (d *DAGResolver) getNodeMap() map[string]Node       { return d.nodeMap }
func (d *DAGResolver) getSegmentMap() map[string]Segment { return d.segmentMap }

func (d *DAGResolver) GetConvergencePoints() map[string]int {
	incomingCounts := d.CountIncomingEdges()
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

func (d *DAGResolver) CountIncomingEdges() map[string]int {
	counts := make(map[string]int)
	nodeMap := d.getNodeMap()

	for _, node := range nodeMap {
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
