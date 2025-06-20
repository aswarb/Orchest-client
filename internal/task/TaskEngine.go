package task

import (
	"context"
	"fmt"
	"io"
	"maps"
	mc "orchest-client/internal/multiClosers"
	wp "orchest-client/internal/workerpool"
	"os/exec"
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
}

func (p ParallelTaskArgs) IsTask() bool { return true }

func GetTaskEngine(resolver *DAGResolver) *TaskEngine {
	engine := TaskEngine{
		resolver:          resolver,
		cmdMap:            make(map[string]*exec.Cmd),
		taskStdinBuffers:  make(map[string]map[string]chan []byte),
		procStdinReaders:  make(map[string]io.ReadCloser),
		procStdoutWriters: make(map[string]io.WriteCloser),
		taskChannelMap:    make(map[string]chan taskCtrlSignal),
	}

	return &engine
}

type TaskEngine struct {
	resolver          *DAGResolver
	cmdMap            map[string]*exec.Cmd
	taskStdinBuffers  map[string]map[string]chan []byte // {receier_uid: {sender_uid: chan []byte}}
	procStdinReaders  map[string]io.ReadCloser          //receiver: io.ReadCloser
	procStdoutWriters map[string]io.WriteCloser         //sender: io.WriteCloser
	taskChannelMap    map[string]chan taskCtrlSignal
}

func (t *TaskEngine) populateCmdMap() {
	for _, node := range t.resolver.GetNodeMap() {
		task, taskIsValid := node.(*Task)
		if !taskIsValid {
			continue
		}
		cmd := exec.Command(task.Executable, task.Args...)

		t.cmdMap[task.GetUid()] = cmd
	}

}

// Creates pipes for adjacent nodes
func (t *TaskEngine) createPipes() {

	t.procStdinReaders = make(map[string]io.ReadCloser)
	t.procStdoutWriters = make(map[string]io.WriteCloser)
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

			if _, arrExists := stdoutEndpoints[receivingUid]; !arrExists {
				stdoutEndpoints[receivingUid] = []io.WriteCloser{}
			}

			pipeReader, pipeWriter := io.Pipe()
			if receivingTask.ReadStdin {
				stdinEndpoints[receivingUid] = append(stdinEndpoints[receivingUid], pipeReader)
			}
			if sendingTask.GiveStdout {
				stdoutEndpoints[sendingUid] = append(stdoutEndpoints[sendingUid], pipeWriter)
			}
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
				t.procStdinReaders[sendingUid] = mr
			} else if len(stdinReaders) == 1 {
				cmd.Stdin = stdinReaders[0]
				t.procStdinReaders[sendingUid] = stdinReaders[0]
			}
		}

		if stdoutWriters, arrExists := stdoutEndpoints[sendingUid]; arrExists {
			if len(stdoutWriters) > 1 {
				mw := mc.MakeMultiWriteCloser(stdoutWriters...)
				cmd.Stdout = mw
				t.procStdoutWriters[sendingUid] = mw
			} else if len(stdoutWriters) == 1 {
				cmd.Stdout = stdoutWriters[0]
				t.procStdoutWriters[sendingUid] = stdoutWriters[0]
			}
		}
	}
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

func (t *TaskEngine) singleTaskWithPipeRoutine(outputChan chan statusUpdate, inputChan chan taskCtrlSignal, taskUid string, cmd *exec.Cmd) {
	outputChan <- statusUpdate{uid: taskUid, started: true, finished: false}
	err := cmd.Run()
	fmt.Println(taskUid, err)
	outputChan <- statusUpdate{uid: taskUid, started: true, finished: true}

	for range 2 {
		s := <-inputChan
		if s == stdout_msg {
			if stdoutPipe, exists := t.procStdoutWriters[taskUid]; exists {
				stdoutPipe.Close()
			}
		}
		if s == stdin_msg {
			if stdinPipe, exists := t.procStdinReaders[taskUid]; exists {
				stdinPipe.Close()
			}
		}
	}
}

func (t *TaskEngine) ExecuteTasksInOrder(ctx context.Context) {
	t.populateCmdMap()
	t.createPipes()
	orderedNodes := t.resolver.GetLinearOrder()
	exploredNodes := make(map[string]struct{})

	outputChan := make(chan statusUpdate, 3)
	for _, n := range orderedNodes {
		t.taskChannelMap[n.GetUid()] = make(chan taskCtrlSignal, 2)
	}
	taskPipeManager := func(inputChan chan statusUpdate) {
		runningTasks := make(map[string]bool)
		for {
			status := <-inputChan
			taskIsRunning := status.started && !status.finished
			if _, exists := runningTasks[status.uid]; exists {
				runningTasks[status.uid] = taskIsRunning
			}
			if channel, exists := t.taskChannelMap[status.uid]; exists && !taskIsRunning {
				channel <- stdin_msg
				channel <- stdout_msg
				break
			}
		}
	}
	go taskPipeManager(outputChan)

	for _, node := range orderedNodes {
		fmt.Println(node)
		if _, nodeIsExplored := exploredNodes[node.GetUid()]; nodeIsExplored {
			fmt.Println(node, "explored")
			continue
		}
		task := node.(*Task)
		segments, segmentSetExists := t.resolver.GetSegments(task.GetUid())
		segmentSetExists = segmentSetExists || len(segments) > 0
		if cmd, cmdExists := t.cmdMap[task.GetUid()]; cmdExists && !segmentSetExists {
			if task.ReadStdin || task.GiveStdout {
				if _, bufferExists := t.taskStdinBuffers[task.GetUid()]; bufferExists && task.ReadStdin {
					// Function to read data from a buffer and put it into stdin:
					go t.stdinChannelConsumerFunc(task.GetUid(), ctx)
				}
				taskInputChan, _ := t.taskChannelMap[task.GetUid()]
				go t.singleTaskWithPipeRoutine(outputChan, taskInputChan, task.GetUid(), cmd)

			} else {
				err := cmd.Run() //blocking
				fmt.Println(task.GetUid(), err)
			}
			exploredNodes[task.GetUid()] = struct{}{}
		} else if cmdExists && segmentSetExists {
			// segment just means parallel block for now, so just start the parallel task
			var segmentUid string
			// segments list should only be of len = 1
			for k := range segments {
				segmentUid = k
				break
			}
			fmt.Println("starting segment", segmentUid)
			uids := t.executeParallelTask(segmentUid, ctx)
			for _, uid := range uids {
				exploredNodes[uid] = struct{}{}
			}
		}
	}
}

func (t *TaskEngine) executeParallelTask(segmentUid string, ctx context.Context) []string {

	fmt.Println("starting execution for segment", segmentUid)
	ctx, cancelFunc := context.WithCancel(context.Background())

	segment, _ := t.resolver.GetSegment(segmentUid)
	linearOrderedTasks := t.resolver.GetLinearOrderFromSegment(segmentUid)
	incomingCounts := t.resolver.CountIncomingEdges(nil)
	zeroIncomingCountsFilter := func(k string, v int) bool { return v == 0 }
	startNodes := maps.Keys(FilterMap(incomingCounts, zeroIncomingCountsFilter))
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

	outputChannel := make(chan packet, 3)
	signalChannel := make(chan struct{}, 3)

	fmt.Println("Channels made")
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

	fmt.Println("Starting handleRequestRoutine")

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
					t.taskChannelMap[uid] <- stdin_msg
					t.taskChannelMap[uid] <- stdout_msg
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
					fmt.Println("Buffering:", data, "from", sender)
					go pushDataToChannel(data, channel, ctx)

				}
			}
		}
	}
	go handleRequestRoutine()

	for uid := range startNodes {
		t.taskChannelMap[uid] = make(chan taskCtrlSignal, 3)

		args := ParallelTaskArgs{
			startUid:   uid,
			currentUid: uid,
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
	}
	workerpool.StartWork(ctx)
	<-signalChannel // B

	fmt.Println("Stop Signal received")
	cancelFunc()
	return slices.Collect(maps.Keys(finishedTasks))
}

func (t *TaskEngine) onParallelExecute(w *wp.Worker, wt *wp.WorkerTask) error {
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
	//defer readCloser.Close()
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
	//defer writeCloser.Close()

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
}
func (t *TaskEngine) onParallelError(w *wp.Worker, wt *wp.WorkerTask, err error) {}
