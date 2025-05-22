package taskParser

import (
	"io"
	"maps"
)

type DAGNode struct {
	uid      string
	nextUids []string
	prevUids []string // Prev only exists to make traversal easier, the nodes only execute from prev -> node -> next -> etc...
}

func (d DAGNode) GetUid() string        { return d.uid }
func (d DAGNode) GetNextUids() []string { return d.nextUids }
func (d DAGNode) GetPrevUids() []string { return d.prevUids }

type TaskManager struct {
	tasks map[string]Task
	graph map[string]DAGNode
}

func (tm *TaskManager) executeTaskProcess() {}

func (tm *TaskManager) GetTaskMap() map[string]Task {
	return tm.tasks
}

func (tm *TaskManager) GetExecutionGraphMap() map[string]DAGNode {
	return tm.graph
}
func (tm *TaskManager) StartTaskChain() {

}

func (tm *TaskManager) CountIncomingEdges() map[string]int {
	incomingEdgeCounts := make(map[string]int)
	for _, task := range tm.GetExecutionGraphMap() {
		uid := task.GetUid()
		_, ok := incomingEdgeCounts[uid]
		if !ok {
			incomingEdgeCounts[uid] = 0
		}
		incomingEdgeCounts[uid]++
	}

	return incomingEdgeCounts
}

func (tm *TaskManager) GetTaskSequence() []Task {
	orderedTasks := []Task{}

	incomingEdgeCounts := tm.CountIncomingEdges()
	zeroDegreeNodesSet := make(map[Task]struct{})
	for uid, count := range incomingEdgeCounts {
		task := tm.GetTaskByUid(uid)
		if count == 0 && task != nil {
			zeroDegreeNodesSet[task] = struct{}{}
		}
	}

	// Kahn's Algorithm: https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm
	inEdgesCounts := make(map[string]int)
	maps.Copy(inEdgesCounts, incomingEdgeCounts)
	for len(zeroDegreeNodesSet) > 0 {
		var pointer Task
		for k := range zeroDegreeNodesSet {
			pointer = k
			break
		}

		delete(zeroDegreeNodesSet, pointer)
		orderedTasks = append(orderedTasks, pointer)
		node := tm.getExecutionGraphNode(pointer.GetUid())
		for _, next := range node.GetNextUids() {
			_, ok := inEdgesCounts[next]
			if ok {
				inEdgesCounts[next]--
			}
			if inEdgesCounts[next] <= 0 {
				zeroDegreeNodesSet[tm.GetTaskByUid(next)] = struct{}{}
			}
		}
	}
	return orderedTasks

}

func (tm *TaskManager) GetTaskByUid(uid string) Task {
	task, ok := tm.GetTaskMap()[uid]

	if ok {
		return task
	}
	return nil
}

func (tm *TaskManager) getExecutionGraphNode(uid string) *DAGNode {
	node, ok := tm.GetExecutionGraphMap()[uid]

	if ok {
		return &node
	}
	return nil

}

func (tm *TaskManager) SetGraph([]Task, []DAGNode) {

}

// Only creates pipes from this node's stdout to each of its next nodes' stdin
func (tm *TaskManager) CreateForwardPipes(n1 DAGNode) {

	destWriters := []io.Writer{}

	for _, nextNode := range n1.GetNextUids() {
		task := tm.GetTaskByUid(nextNode)
		if task.GetReadStdin() && task.GetGiveStdout() {
			pipeRead, pipeWrite := io.Pipe()
			destWriters = append(destWriters, pipeWrite)
			task.SetStdin(pipeRead)
		}
	}

	t1 := tm.GetTaskByUid(n1.GetUid())
	if t1.GetGiveStdout() && len(destWriters) > 0 {
		multiWriter := io.MultiWriter(destWriters...)
		t1.SetStdout(multiWriter)
	}

}
