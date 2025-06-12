package task

import (
	"context"
	"io"
	mc "orchest-client/internal/multiClosers"
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

type ParallelTaskArgs struct {
	startUid   string
	currentUid string
	segmentUid string
	endUids    []string
	outputChan chan packet
	context    context.Context
}

func (p ParallelTaskArgs) IsTask() bool { return true }

type TaskEngine struct {
	resolver          *DAGResolver
	cmdMap            map[string]*exec.Cmd
	taskStdinBuffers  map[string]map[string]chan []byte // {receier_uid: {sender_uid: chan []byte}}
	procStdinReaders  map[string]io.ReadCloser          //receiver: io.ReadCloser
	procStdoutWriters map[string]io.WriteCloser         //sender: io.WriteCloser
}

// Creates pipes for adjacent nodes
func (t *TaskEngine) createPipes() {

	stdinEndpoints := make(map[string][]io.ReadCloser)
	stdoutEndpoints := make(map[string][]io.WriteCloser)

	allNodes := t.resolver.GetLinearOrder()

	for _, node := range allNodes {
		sendingTask, ok := node.(*Task)
		if !ok {
			continue
		}
		sendingUid := sendingTask.GetUid()

		if _, arrExists := stdinEndpoints[sendingUid]; !arrExists {
			stdinEndpoints[sendingUid] = []io.ReadCloser{}
		}

		nextUids := sendingTask.GetNext()
		for _, uid := range nextUids {
			receivingNode, nodeExists := t.resolver.GetNode(uid)
			if !nodeExists {
				continue
			}
			receivingTask, ok := receivingNode.(*Task)
			if !ok {
				continue
			}
			receivingUid := receivingTask.GetUid()

			if _, arrExists := stdinEndpoints[sendingUid]; !arrExists {
				stdoutEndpoints[receivingUid] = []io.WriteCloser{}
			}

			pipeReader, pipeWriter := io.Pipe()
			stdinEndpoints[receivingUid] = append(stdinEndpoints[receivingUid], pipeReader)
			stdoutEndpoints[sendingUid] = append(stdoutEndpoints[sendingUid], pipeWriter)
		}
	}

	for _, node := range allNodes {
		task, ok := node.(*Task)
		if !ok {
			continue
		}
		sendingUid := task.GetUid()

		cmd, cmdExists := t.cmdMap[sendingUid]

		if !cmdExists {
			continue
		}

		if stdinReaders, arrExists := stdinEndpoints[sendingUid]; arrExists {
			if len(stdinReaders) > 1 {
				mr := mc.MakeMultiReadCloser(stdinReaders...)
				cmd.Stdin = mr
			} else if len(stdinReaders) == 1 {
				cmd.Stdin = stdinReaders[0]
			}
		}
		if stdoutWriters, arrExists := stdoutEndpoints[sendingUid]; arrExists {
			if len(stdoutWriters) > 1 {
				mw := mc.MakeMultiWriteCloser(stdoutWriters...)
				cmd.Stdout = mw
			} else if len(stdoutWriters) == 1 {
				cmd.Stdout = stdoutWriters[0]
			}
		}
	}
}

// TODO Needs finishing
func (t *TaskEngine) ExecuteTasksInOrder(ctx context.Context) {
	t.createPipes()
	orderedNodes := t.resolver.GetLinearOrder()
	for _, node := range orderedNodes {
		task := node.(*Task)
		segments, segmentSetExists := t.resolver.GetSegments(task.GetUid())
		segmentSetExists = segmentSetExists || len(segments) > 0
		if cmd, cmdExists := t.cmdMap[task.GetUid()]; cmdExists && !segmentSetExists {
			if task.ReadStdin || task.GiveStdout {
				if task.ReadStdin {
					// Function to read data from a buffer and put it into stdin:
					go t.stdinChannelConsumerFunc(task.GetUid(), ctx)
				}
				cmd.Start() //non-blocking
			} else {
				cmd.Run() //blocking
			}
		} else if cmdExists && segmentSetExists {
			// segment just means parallel block for now, so just start the parallel task
			var segmentUid string
			// segments list should only be of len = 1
			for k := range segments {
				segmentUid = k
				break
			}
			t.executeParallelTask(segmentUid, ctx)
		}
	}
}

func (t *TaskEngine) executeParallelTask(segmentUid string, ctx context.Context) {
	ctx, cancelFunc := context.WithCancel(context.Background())

	segment, _ := t.resolver.GetSegment(segmentUid)
	linearOrderedTasks := t.resolver.GetLinearOrderFromSegment(segmentUid)

	incomingCounts := t.resolver.CountIncomingEdges(nil)
	branchCount := 0

	for _, count := range incomingCounts {
		if count == 0 {
			branchCount++
		}
	}

	for _, t := range linearOrderedTasks {
		uid := t.GetUid()
		if count, ok := incomingCounts[uid]; ok && count > 1 {
			branchCount -= count - 1
		}

		if next := t.GetNext(); len(next) > 1 {
			branchCount += len(next) - 1
		}
	}

	workerpool := wp.MakeWorkerPool(ctx)
	workerpool.AddWorkers(uint(branchCount))
	startedTasks := make(map[string]struct{})
	finishedTasks := make(map[string]struct{})

	outputChannel := make(chan packet)
	signalChannel := make(chan struct{})

	for k, v := range incomingCounts {
		if v < 2 {
			delete(incomingCounts, k)
		}
	}

	pushDataToChannel := func(data []byte, channel chan []byte, ctx context.Context) {
		select {
		case channel <- data:
		case <-ctx.Done():
			return
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
					if len(finishedTasks) == len(linearOrderedTasks) {
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
					bufferPacket := output.(*bufferDataPacket)
					data := bufferPacket.getData()
					sender := bufferPacket.getSender()
					receiver := bufferPacket.getTarget()

					if _, ok := t.taskStdinBuffers[receiver]; !ok {
						t.taskStdinBuffers[receiver] = make(map[string]chan []byte)
					}
					if _, ok := t.taskStdinBuffers[receiver][sender]; !ok {
						t.taskStdinBuffers[receiver][sender] = make(chan []byte)
					}
					channel, _ := t.taskStdinBuffers[receiver][sender]
					go pushDataToChannel(data, channel, ctx)
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
		context:    ctx,
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
	task := node.(*Task)
	//segment, _ := t.resolver.GetSegment(args.segmentUid)

	args.currentUid = task.GetUid()

	cmd, _ := t.cmdMap[task.GetUid()]

	args.outputChan <- &taskStartedPacket{uid: task.GetUid()}
	if task.ReadStdin || task.GiveStdout {
		// Function to read data from stdout and put it into a buffer:
		go t.stdoutConsumerFunc(task.GetUid(), args.outputChan, args.context)
		cmd.Start()
	} else {
		cmd.Run()

	}

	return nil
}

// Designed to run as a routine
func (t *TaskEngine) stdoutConsumerFunc(sendingUid string, outputChan chan packet, ctx context.Context) {
	// Func needs to properly duplicate output to all next nodes
	readCloser, readerExists := t.procStdinReaders[sendingUid]
	if !readerExists {
		return
	}
	node, _ := t.resolver.GetNode(sendingUid)
	task := node.(*Task)
	nextUids := task.GetNext()

	defer readCloser.Close()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			buf := make([]byte, 4096)
			n, err := readCloser.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				return
			}
			for _, receivingUid := range nextUids {
				data := make([]byte, n)
				if err == io.EOF {
					data = nil
				} else {
					copy(data, buf[:n])
				}
				bufferPacket := &bufferDataPacket{thisUid: receivingUid, targetUid: sendingUid, data: data}
				outputChan <- bufferPacket
			}

		}
	}
}

func (t *TaskEngine) stdinChannelConsumerFunc(receivingUid string, ctx context.Context) {
	writeCloser, writerExists := t.procStdoutWriters[receivingUid]
	if !writerExists {
		return
	}
	defer writeCloser.Close()

	exhaustedBuffers := make(map[string]struct{})
	for {
		bufferSet, bufferSetExists := t.taskStdinBuffers[receivingUid]
		if !bufferSetExists {
			return
		}
		allExhausted := true // Assume no data in buffer
		for senderUid, buffer := range bufferSet {
			if _, ok := exhaustedBuffers[senderUid]; ok {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case data, ok := <-buffer:
				if !ok || data == nil {
					exhaustedBuffers[senderUid] = struct{}{}
					continue
				}
				allExhausted = false // Loop again if you were able to get any data from the buffer
				_, err := writeCloser.Write(data)
				if err != nil {
					return
				}
			}
		}
		if allExhausted {
			break
		}
	}
}

func (t *TaskEngine) onParallelComplete(w *wp.Worker, wt *wp.WorkerTask) {

	incomingEdgeCounts := t.resolver.CountIncomingEdges(nil)
	filterFunc := func(k string, v int) bool { return v > 1 }
	conergencePoints := FilterMap(incomingEdgeCounts, filterFunc)
	args := wt.Args.(*ParallelTaskArgs)
	node, _ := t.resolver.GetNode(args.startUid)
	nextNodes := node.GetNext()

	if len(nextNodes) > 1 {
		for _, uid := range nextNodes {
			args := ParallelTaskArgs{
				startUid:   args.startUid,
				currentUid: uid,
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

			_, isConvergence := conergencePoints[uid]

			if isConvergence {
				replyChannel := make(chan bool)
				req := proceedRequestPacket{
					thisUid:      node.GetUid(),
					targetUid:    uid,
					replyChannel: replyChannel,
				}
				args.outputChan <- &req
				response := <-replyChannel
				if response == true {
					w.GetTaskChan() <- &nextTask
				}
			} else {
				w.GetTaskChan() <- &nextTask
			}
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

}
func (t *TaskEngine) onParallelError(w *wp.Worker, wt *wp.WorkerTask, err error) {}
