package task

import (
	"context"
	wp "orchest-client/internal/workerpool"
	"os/exec"
)

type packet interface {
	isPacket() bool
	getSender() string
}
type taskStartedPacket struct {
	uid string
}

func (p *taskStartedPacket) isPacket() bool    { return true }
func (p *taskStartedPacket) getSender() string { return p.uid }

type taskCompletePacket struct {
	uid string
}

func (p *taskCompletePacket) isPacket() bool    { return true }
func (p *taskCompletePacket) getSender() string { return p.uid }

type proceedRequestPacket struct {
	thisUid      string
	targetUid    string
	replyChannel chan bool // False for deny request, True for request granted
}

func (p *proceedRequestPacket) isPacket() bool             { return true }
func (p *proceedRequestPacket) getSender() string          { return p.thisUid }
func (p *proceedRequestPacket) getTarget() string          { return p.targetUid }
func (p *proceedRequestPacket) getReplyChannel() chan bool { return p.replyChannel }

type bufferDataPacket struct {
	thisUid   string
	targetUid string
	data      []byte
}

func (p *bufferDataPacket) isPacket() bool    { return true }
func (p *bufferDataPacket) getSender() string { return p.thisUid }
func (p *bufferDataPacket) getTarget() string { return p.targetUid }
func (p *bufferDataPacket) getData() []byte   { return p.data }

type TaskEngine struct {
	resolver         *DAGResolver
	cmdMap           map[string]*exec.Cmd
	taskStdinBuffers map[string]chan []byte
}
type ParallelTaskArgs struct {
	startUid   string
	currentUid string
	segmentUid string
	endUids    []string
	outputChan chan packet
}

func (p ParallelTaskArgs) IsTask() bool { return true }

// Creates pipes for adjacent nodes
func (t *TaskEngine) createPipes() {
}

func (t *TaskEngine) ExecuteTasksInOrder() {
	// Do something ...
}

func (t *TaskEngine) executeSingleTask(taskUid string) {
	// Do something ...
}

func (t *TaskEngine) executeParallelTask(segmentUid string) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	segment, _ := t.resolver.GetSegment(segmentUid)
	someTarget := 10 // Needs to be gotten from segment task count

	workerpool := wp.MakeWorkerPool(ctx)
	workerpool.AddWorkers(uint(1))
	startedTasks := make(map[string]struct{})
	finishedTasks := make(map[string]struct{})

	outputChannel := make(chan packet)
	signalChannel := make(chan struct{})

	incomingCounts := t.resolver.CountIncomingEdges()
	for k, v := range incomingCounts {
		if v < 2 {
			delete(incomingCounts, k)
		}
	}

	handleRequestRoutine := func() {
		for {
			select {
			case <-ctx.Done():
				break
			case output := <-outputChannel:
				switch p := output.(type) {
				case *taskStartedPacket:
					uid := p.getSender()
					startedTasks[uid] = struct{}{}
				case *taskCompletePacket:
					uid := p.getSender()
					finishedTasks[uid] = struct{}{}
					if len(finishedTasks) == someTarget {
						signalChannel <- struct{}{}
					}
				case *proceedRequestPacket:
					targetUid := p.getTarget()
					replyChannel := p.getReplyChannel()
					_, ok := startedTasks[targetUid]
					if ok {
						replyChannel <- true
					} else {
						replyChannel <- false
					}
				case *bufferDataPacket:
					// Do something ...
				}
			}
		}
	}
	go handleRequestRoutine()

	args := ParallelTaskArgs{
		startUid:   "",
		currentUid: "",
		segmentUid: segment.GetUid(),
		endUids:    segment.GetEndpointUids(),
		outputChan: outputChannel,
	}
	parallelExecuteTask := wp.WorkerTask{
		Args:       args,
		Execute:    t.onParallelExecute,
		OnComplete: t.onParallelComplete,
		OnError:    t.onParallelError,
	}
	workerpool.AddTask(&parallelExecuteTask)

	<-signalChannel // Blocks until correct number of finished tasks completed

	cancelFunc()
}

func (t *TaskEngine) onParallelExecute(w *wp.Worker, wt *wp.WorkerTask) error {
	args := wt.Args.(*ParallelTaskArgs)
	node, _ := t.resolver.GetNode(args.startUid)
	args.currentUid = node.GetUid()

	nextNodes := node.GetNext()
	if len(nextNodes) > 1 {
		for i, uid := range nextNodes {
			// Do something...
		}
	} else if len(nextNodes) == 1 {

		args := ParallelTaskArgs{
			startUid:   args.startUid,
			currentUid: nextNodes[0],
			segmentUid: args.segmentUid,
			endUids:    args.endUids,
			outputChan: args.outputChan,
		}

		nextTask := wp.WorkerTask{
			Args:       args,
			Execute:    t.onParallelExecute,
			OnComplete: t.onParallelComplete,
			OnError:    t.onParallelError,
		}

		taskChannel := w.GetTaskChan()
		taskChannel <- &nextTask
	}

	completePacket := taskCompletePacket{uid: args.currentUid}
	args.outputChan <- &completePacket

	return nil
}
func (t *TaskEngine) onParallelComplete(w *wp.Worker, wt *wp.WorkerTask)         {}
func (t *TaskEngine) onParallelError(w *wp.Worker, wt *wp.WorkerTask, err error) {}
