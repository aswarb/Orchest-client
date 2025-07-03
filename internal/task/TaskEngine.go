package task

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	mc "orchest-client/internal/multiClosers"
	wp "orchest-client/internal/workerpool"
	"slices"
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

type ParallelTaskArgs struct {
	startUid   string
	currentUid string
	outputChan chan packet
}

func (p ParallelTaskArgs) IsTask() bool { return true }

func GetTaskEngine(resolver *DAGResolver) *TaskEngine {
	engine := TaskEngine{
		resolver:       resolver,
		cmdMap:         make(map[string]*CmdWrapper),
		taskChannelMap: make(map[string]chan taskCtrlSignal),
	}

	return &engine
}

type TaskEngine struct {
	resolver       *DAGResolver
	cmdMap         map[string]*CmdWrapper
	taskChannelMap map[string]chan taskCtrlSignal
}

func (t *TaskEngine) createPipesNew() {
	incomingPipes := make(map[string][]io.ReadCloser)
	outgoingPipes := make(map[string][]io.WriteCloser)

	wrappedCmds := make(map[string]*CmdWrapper)

	allNodes := t.resolver.GetLinearOrder()
	fmt.Println("createPipesNew allNodes", allNodes)
	for _, node := range allNodes {
		uid := node.GetUid()
		fmt.Println("createPipesNew trying to create pipe end for", uid)
		prevTask := node.(*Task)
		nextUids := node.GetNext()
		for _, nUid := range nextUids {
			nextNode, nextNodeExists := t.resolver.GetNode(nUid)
			if !nextNodeExists {
				continue
			}
			nextTask := nextNode.(*Task)
			pipeReader, pipeWriter := io.Pipe()
			// Assume stdin-stdout pairs have already been validated
			// Note: this means nothing should take on stdin without at least 1 task pointing to it that gives stdout
			if nextTask.ReadStdin && prevTask.GiveStdout {

				if _, exists := incomingPipes[nUid]; !exists {
					incomingPipes[nUid] = []io.ReadCloser{}
				}

				if _, exists := outgoingPipes[uid]; !exists {
					outgoingPipes[uid] = []io.WriteCloser{}
				}
				fmt.Println("createPipesNew storing pipe ends from ", uid, "to", nUid)
				incomingPipes[nUid] = append(incomingPipes[nUid], pipeReader)
				outgoingPipes[uid] = append(outgoingPipes[uid], pipeWriter)
			}
		}
	}

	for _, node := range allNodes {
		uid := node.GetUid()

		fmt.Println("createPipesNew Getting pipe ends for ", uid)
		fmt.Println("incomingPipes:", incomingPipes)
		incoming, _ := incomingPipes[uid]
		fmt.Println("outgoingPipes:", outgoingPipes)
		outgoing, _ := outgoingPipes[uid]
		if len(incoming) > 1 {
			fmt.Println("createPipesNew binding many incoming ends for ", uid)
			incomingPipes[uid] = []io.ReadCloser{mc.MakeMultiReadCloser(incoming...)}
		}
		if len(outgoing) > 1 {
			fmt.Println("createPipesNew binding many outgoing ends for ", uid)
			outgoingPipes[uid] = []io.WriteCloser{mc.MakeMultiWriteCloser(outgoing...)}
		}
	}

	for _, node := range allNodes {
		uid := node.GetUid()
		fmt.Println("createPipesNew trying to wrap endpoints for", uid)
		task := node.(*Task)
		outgoing, outgoingExists := outgoingPipes[uid]
		if !outgoingExists {
			continue
		}
		outpoint := outgoing[0]

		incoming, incomingExists := incomingPipes[uid]
		if !incomingExists {
			continue
		}

		inpoint := incoming[0]

		wrapper := CreateCmdWrapper(task.Executable, task.Args, inpoint, outpoint)
		wrappedCmds[uid] = wrapper
	}
	t.cmdMap = wrappedCmds
	fmt.Println(wrappedCmds)

}

type statusUpdate struct {
	uid      string
	started  bool
	finished bool
}
type taskCtrlSignal string

const (
	stdin_msg  taskCtrlSignal = "STDIN"
	stdout_msg taskCtrlSignal = "STDOUT"
)

func (t *TaskEngine) ExecuteTasksInOrder(ctx context.Context) {
	t.createPipesNew()
	t.executeParallelTask(ctx)
}

func (t *TaskEngine) executeParallelTask(ctx context.Context) []string {

	ctx, cancelFunc := context.WithCancel(context.Background())

	linearOrderedTasks := t.resolver.GetLinearOrder()
	// Only interested in incoming nodes for nodes in this segment
	incomingCounts := t.resolver.CountIncomingEdges(linearOrderedTasks)
	zeroIncomingCountsFilter := func(k string, v int) bool { return v == 0 }
	startNodes := maps.Keys(FilterMap(incomingCounts, zeroIncomingCountsFilter))

	workerpool := wp.MakeWorkerPool(ctx)
	// one worker per task, tasks won't execute until they're queued,
	// so waiting workers are find to be idle
	workerpool.AddWorkers(uint(len(incomingCounts)))
	startedTasks := make(map[string]struct{})
	finishedTasks := make(map[string]struct{})

	outputChannel := make(chan packet, 3)
	signalChannel := make(chan struct{}, 5)

	fmt.Println("Channels made")

	fmt.Println("Starting handleRequestRoutine")
	taskCount := len(incomingCounts)
	handleRequestRoutine := func() {
		for {
			select {
			case <-ctx.Done():
				return
			case output := <-outputChannel:
				switch p := output.(type) {
				case *taskStartedPacket:
					uid := p.getSender()
					startedTasks[uid] = struct{}{}
					node, _ := t.resolver.GetNode(uid)
					fmt.Println(incomingCounts)
					for _, nextUid := range node.GetNext() {
						if count, exists := incomingCounts[nextUid]; exists {
							fmt.Println("parallelExecuteTask-manager", "Starting task", nextUid)
							if count > 0 {
								incomingCounts[nextUid]--
								count = incomingCounts[nextUid]
							}
							fmt.Println(incomingCounts)
							if count == 0 {
								args := ParallelTaskArgs{
									startUid:   uid,
									currentUid: nextUid,
									outputChan: outputChannel,
								}
								parallelExecuteTask := wp.WorkerTask{
									Args:       args,
									Execute:    t.onParallelExecute,
									OnComplete: t.onParallelComplete,
									OnError:    t.onParallelError,
								}
								t.taskChannelMap[nextUid] = make(chan taskCtrlSignal)
								workerpool.AddTask(&parallelExecuteTask)
								delete(incomingCounts, nextUid)
							}
						}
					}
				case *taskCompletePacket:
					uid := p.getSender()
					fmt.Println("taskCompletePacket received from", uid)
					fmt.Println(t.taskChannelMap)
					t.taskChannelMap[uid] <- stdin_msg
					t.taskChannelMap[uid] <- stdout_msg
					finishedTasks[uid] = struct{}{}
					if len(finishedTasks) == taskCount {
						// Send signal to stop blocking of the main parallel execute function
						signalChannel <- struct{}{}
						return
					}
				}
			}
		}
	}

	for uid := range startNodes {
		t.taskChannelMap[uid] = make(chan taskCtrlSignal, 3)

		args := ParallelTaskArgs{
			startUid:   uid,
			currentUid: uid,
			outputChan: outputChannel,
		}
		parallelExecuteTask := wp.WorkerTask{
			Args:       args,
			Execute:    t.onParallelExecute,
			OnComplete: t.onParallelComplete,
			OnError:    t.onParallelError,
		}

		workerpool.AddTask(&parallelExecuteTask)
		delete(incomingCounts, uid)
	}
	go handleRequestRoutine()

	workerpool.StartWork(ctx)
	<-signalChannel // B

	cancelFunc()
	fmt.Println("Parallel Segment finished", slices.Collect(maps.Keys(finishedTasks)))
	return slices.Collect(maps.Keys(finishedTasks))
}

type TaskNotFoundError struct {
	Uid string
}

func (e *TaskNotFoundError) Error() string {
	return fmt.Sprint("Task not found, uidL", e.Uid)
}

func (t *TaskEngine) onParallelExecute(w *wp.Worker, wt *wp.WorkerTask) error {
	// start task
	args := wt.Args.(ParallelTaskArgs)

	node, nodeExists := t.resolver.GetNode(args.currentUid)
	if !nodeExists {
		return &TaskNotFoundError{Uid: args.currentUid}
	}
	task := node.(*Task)
	cmd, cmdExists := t.cmdMap[task.GetUid()]
	fmt.Println("onParallelExecute trying to start", task.GetUid())
	if cmdExists && (task.GiveStdout || task.ReadStdin) {
		fmt.Println("onParallelExecute trying to start", task.GetUid(), "with stdin/stdout")
		go func() {
			args.outputChan <- &taskStartedPacket{uid: args.currentUid}

			err := cmd.ExecuteBlocking()
			fmt.Println("onParallelExecute-anon", task.GetUid(), "Finished with err:", err)
			args.outputChan <- &taskCompletePacket{uid: args.currentUid}
		}()

	} else if cmdExists {
		fmt.Println("onParallelExecute trying to start", task.GetUid(), "without stdin/stdout")
		args.outputChan <- &taskStartedPacket{uid: args.currentUid}
		err := cmd.ExecuteBlocking()
		fmt.Println("onParallelExecute", task.GetUid(), err)
		args.outputChan <- &taskCompletePacket{uid: args.currentUid}
	}
	fmt.Println("currentUid", args.currentUid)
	return nil
}

func (t *TaskEngine) onParallelComplete(w *wp.Worker, wt *wp.WorkerTask) {
	args := wt.Args.(ParallelTaskArgs)
	node, _ := t.resolver.GetNode(args.currentUid)
	task, _ := node.(*Task)

	inputChan, _ := t.taskChannelMap[task.uid]

	cmd, _ := t.cmdMap[args.currentUid]
	go func() {
		stdinClosed := false
		stdoutClosed := false

		for !stdinClosed || !stdoutClosed {
			fmt.Println("onParallelComplete-anon", args.currentUid, "Waiting for close signal")
			signal := <-inputChan
			fmt.Println("onParallelComplete-anon", args.currentUid, "signal received", signal)
			if signal == stdin_msg && !stdinClosed {
				err := cmd.CloseStdin()
				fmt.Println(task.GetUid(), ": Closing Stdin", err)
				stdinClosed = true
			}
			if signal == stdout_msg && !stdoutClosed {
				err := cmd.CloseStdout()
				fmt.Println(task.GetUid(), ": Closing Stdout", err)
				stdoutClosed = true
			}
		}
		fmt.Println(task.GetUid(), ": All pipes closed")
	}()

}

func (t *TaskEngine) onParallelError(w *wp.Worker, wt *wp.WorkerTask, err error) {
	// Re-queue task if not ready, log error
	switch e := errors.Unwrap(err); e.(type) {
	case *TaskNotFoundError:
		fmt.Println(e)
	default:
		fmt.Println(e)
	}
}
