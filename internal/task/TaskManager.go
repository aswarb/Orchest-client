package task

import (
	"context"
	"fmt"
	"io"
	"maps"
	wp "orchest-client/internal/workerpool"
	"sync"
)

func FilterMap[K comparable, V any](m map[K]V, isValid func(K, V) bool) map[K]V {
	result := make(map[K]V)
	for k, v := range m {
		if isValid(k, v) {
			result[k] = v
		}
	}
	return result
}

func MapHasKey[K comparable, V any](m map[K]V, key K) bool {
	_, ok := m[key]
	return ok
}

type DAGNode struct {
	uid      string
	nextUids []string
}

func (d DAGNode) GetUid() string        { return d.uid }
func (d DAGNode) GetNextUids() []string { return d.nextUids }

type TaskManager struct {
	tasks map[string]Task
	graph map[string]DAGNode
}

type ParallelTaskArgs struct {
	startUid   string
	currentUid string
}

func (p ParallelTaskArgs) IsTask() bool {
	return true
}

func (tm *TaskManager) executeTask(task Task) {

}

func (tm *TaskManager) BetterExecuteTaskProcess() error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	workerpool := wp.MakeWorkerPool(ctx)
	workerpool.AddWorkers(uint(1))

	sequence := tm.GetTaskSequence()
	startedTasks := make(map[string]struct{})

	for _, task := range sequence {
		giveStdout := task.GetGiveStdout()
		node, ok := tm.GetExecutionGraphNode(task.GetUid())
		if !ok || MapHasKey(startedTasks, task.GetUid()) {
			continue
		}
		switch task.(type) {
		case *SingleTask:
			tm.CreateForwardPipes(*node)
			task.Execute()
			startedTasks[task.GetUid()] = struct{}{}
			for _, nextUid := range node.GetNextUids() {
				next, ok := tm.GetTaskByUid(nextUid)
				if ok && next.GetReadStdin() && !giveStdout {
					return fmt.Errorf("Error. Task %s Tried to read from stdin without Task %s giving stdout", task.GetUid(), next.GetUid())
				}
				if giveStdout && ok && next.GetReadStdin() {
					if !MapHasKey(startedTasks, next.GetUid()) {
						next.Execute()
						startedTasks[next.GetUid()] = struct{}{}
					}
				}
			}

		case *ParallelTask:

			nodeCounts := tm.CountIncomingEdges()
			convergencesCounts := FilterMap(nodeCounts, func(k string, v int) bool { return v >= 1 })

			convergenceUids := []string{}
			convergenceMutex := sync.RWMutex{}

			for k, _ := range convergencesCounts {
				convergenceUids = append(convergenceUids, k)
			}

			onParallelExecute := func(w *wp.Worker, wt *wp.WorkerTask) error {
				args := wt.Args.(*ParallelTaskArgs)
				var node *DAGNode
				node, _ = tm.GetExecutionGraphNode(args.startUid)
				args.currentUid = node.GetUid()
				for len(node.GetNextUids()) != 1 {
					task, _ := tm.GetTaskByUid(node.GetUid())
					task.Execute()
					node, _ = tm.GetExecutionGraphNode(node.GetNextUids()[0])
					args.currentUid = node.GetUid()
				}
				return nil
			}
			onParallelComplete := func(w *wp.Worker, wt *wp.WorkerTask) {

				args := wt.Args.(*ParallelTaskArgs)
				node, _ := tm.GetExecutionGraphNode(args.currentUid)
				nextUids := node.GetNextUids()
				if len(nextUids) < 1 {
					// Leaf node case
				} else if len(nextUids) > 1 {
					// Path diverges case
					newTask := wp.WorkerTask{
						Args:       ParallelTaskArgs{startUid: node.GetUid(), currentUid: ""},
						Execute:    wt.Execute,
						OnComplete: wt.OnComplete,
						OnError:    wt.OnError,
					}
					workerpool.AddTask(&newTask)
				}
			}
			parallelExecutionWpTask := wp.WorkerTask{
				Args:       ParallelTaskArgs{startUid: node.GetUid(), currentUid: ""},
				Execute:    onParallelExecute,
				OnComplete: onParallelComplete,
				OnError:    func(*wp.Worker, *wp.WorkerTask, error) {},
			}

			workerpool.AddTask(&parallelExecutionWpTask)
		}
	}

	cancelFunc()
	return nil
}

func (tm *TaskManager) getNextBranchPoint(startNode *DAGNode) string {
	var nextNode *DAGNode = startNode
	var next []string = startNode.GetNextUids()
	for true {
		nextNode, _ = tm.GetExecutionGraphNode(nextNode.GetUid())

		next = nextNode.GetNextUids()
		if len(next) != 1 {
			return nextNode.GetUid()
		}
	}
	return ""
}

func (tm *TaskManager) ExecuteTaskProcess() error {
	sequence := tm.GetTaskSequence()
	startedTasks := make(map[string]struct{})
	for _, task := range sequence {
		// TODO: Validation that no task reads from stdin without a task writing to stdout
		giveStdout := task.GetGiveStdout()
		node, ok := tm.GetExecutionGraphNode(task.GetUid())
		if !ok {
			continue
		}
		tm.CreateForwardPipes(*node)
		if !MapHasKey(startedTasks, task.GetUid()) {
			fmt.Println("Starting", task.GetUid())
			task.Execute()
			startedTasks[task.GetUid()] = struct{}{}
		}
		for _, nextUid := range node.GetNextUids() {
			next, ok := tm.GetTaskByUid(nextUid)
			if ok && next.GetReadStdin() && !giveStdout {
				return fmt.Errorf("Error. Task %s Tried to read from stdin without Task %s giving stdout", task.GetUid(), next.GetUid())
			}
			if giveStdout && ok && next.GetReadStdin() {
				if !MapHasKey(startedTasks, next.GetUid()) {
					next.Execute()
					startedTasks[next.GetUid()] = struct{}{}
				}
			}
		}
	}
	return nil
}

func (tm *TaskManager) GetTaskMap() map[string]Task {
	return tm.tasks
}

func (tm *TaskManager) GetExecutionGraphMap() map[string]DAGNode {
	return tm.graph
}
func (tm *TaskManager) CountIncomingEdges() map[string]int {
	incomingEdgeCounts := make(map[string]int)
	nodes := tm.GetExecutionGraphMap()
	for _, task := range tm.GetExecutionGraphMap() {
		uid := task.GetUid()
		node := nodes[uid]
		_, taskUidInMap := incomingEdgeCounts[uid]
		if !taskUidInMap {
			incomingEdgeCounts[uid] = 0
		}
		for _, nextUid := range node.GetNextUids() {
			_, nextUidInMap := incomingEdgeCounts[nextUid]
			if nextUidInMap {
				incomingEdgeCounts[nextUid]++
			} else {
				incomingEdgeCounts[nextUid] = 1
			}
		}

	}
	fmt.Println(incomingEdgeCounts)
	return incomingEdgeCounts
}

func (tm *TaskManager) GetTaskSequence() []Task {
	orderedTasks := []Task{}

	incomingEdgeCounts := tm.CountIncomingEdges()
	zeroDegreeNodesSet := make(map[Task]struct{})
	for uid, count := range incomingEdgeCounts {
		task, ok := tm.GetTaskByUid(uid)
		if ok && count == 0 {
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
		node, ok := tm.GetExecutionGraphNode(pointer.GetUid())
		if !ok {
			continue
		}
		for _, next := range node.GetNextUids() {
			_, ok := inEdgesCounts[next]
			if ok {
				inEdgesCounts[next]--
			}
			if inEdgesCounts[next] <= 0 {
				task, ok := tm.GetTaskByUid(next)
				if !ok {
					continue
				}
				zeroDegreeNodesSet[task] = struct{}{}
			}
		}
	}
	return orderedTasks
}

func (tm *TaskManager) GetTaskByUid(uid string) (Task, bool) {
	task, ok := tm.GetTaskMap()[uid]
	return task, ok
}

func (tm *TaskManager) GetExecutionGraphNode(uid string) (*DAGNode, bool) {
	node, ok := tm.GetExecutionGraphMap()[uid]

	return &node, ok

}

func (tm *TaskManager) SetGraph([]Task, []DAGNode) {

}

// Only creates pipes from this node's stdout to each of its next nodes' stdin
func (tm *TaskManager) CreateForwardPipes(n1 DAGNode) {
	destWriters := []io.Writer{}

	for _, nextNode := range n1.GetNextUids() {
		task, ok := tm.GetTaskByUid(nextNode)
		if !ok {
			continue
		}
		if task.GetReadStdin() && task.GetGiveStdout() {
			pipeRead, pipeWrite := io.Pipe()
			destWriters = append(destWriters, pipeWrite)
			task.SetStdin(pipeRead)
		}
	}

	t1, ok := tm.GetTaskByUid(n1.GetUid())
	if ok && t1.GetGiveStdout() && len(destWriters) > 0 {
		multiWriter := io.MultiWriter(destWriters...)
		t1.SetStdout(multiWriter)
	}
}

func GetTaskManagerFromToml(arr []TomlTask) TaskManager {

	tasks := make(map[string]Task)
	graph := make(map[string]DAGNode)

	for _, el := range arr {
		var task Task
		switch t := el.(type) {
		case *SingleTomlTask:
			task = GetSingleTask(
				t.Name,
				t.Command,
				t.Args,
				t.Uid,
				uint64(t.Timeout),
				uint64(t.Delay),
				t.GiveStdout,
				t.ReadStdin,
			)
		case *ParallelTomlTask:
			//
		default:
			continue
		}
		tasks[el.GetUid()] = task
		graph[el.GetUid()] = DAGNode{uid: el.GetUid(), nextUids: el.GetNext()}
	}

	tm := TaskManager{
		tasks: tasks,
		graph: graph,
	}

	return tm
}
