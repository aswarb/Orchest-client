package task

import (
	"context"
	wp "orchest-client/internal/workerpool"
	"os/exec"
)

type TaskEngine struct {
	resolver *DAGResolver
	cmdMap   map[string]*exec.Cmd
}
type ParallelTaskArgs struct {
	startUid   string
	currentUid string
}

func (p ParallelTaskArgs) IsTask() bool {

	return true

}

// Creates pipes for adjacent nodes
func (t *TaskEngine) createPipes() {
}

func (t *TaskEngine) executeParallelTask(segmentUid string) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	segment, _ := t.resolver.GetSegment(segmentUid)

	workerpool := wp.MakeWorkerPool(ctx)
	workerpool.AddWorkers(uint(1))
	startedTasks := make(map[string]struct{})

	cancelFunc()
}
func (t *TaskEngine) onParallelExecute(w *wp.Worker, wt *wp.WorkerTask) error {
	args := wt.Args.(*ParallelTaskArgs)
	node, _ := t.resolver.GetNode(args.startUid)
	args.currentUid = node.GetUid()

	return nil
}
func (t *TaskEngine) onParallelComplete(w *wp.Worker, wt *wp.WorkerTask) {}
func (t *TaskEngine) onParallelError(w *wp.Worker, wt *wp.WorkerTask)    {}
