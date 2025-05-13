package api

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"
)

type TcpListener struct {
	BaseListener
	localAddr   string
	workerCount uint
	ctx         context.Context
	cancelFunc  func()
}

func onCompleteFunc(w *Worker, task *WorkerTask)       {}
func onErrorFunc(w *Worker, task *WorkerTask, e error) {}

func (listener *TcpListener) OpenConnection() {

	workerpool := MakeWorkerPool(listener.ctx)
	workerpool.AddWorkers(listener.workerCount)

	go func() {
		netListener, setupErr := getTcpListener(listener.ctx, listener.localAddr, listener.port)

		workerpool.StartWork(listener.ctx)
		if setupErr != nil {
			fmt.Println(setupErr)
		} else {
			fmt.Println(fmt.Sprintf("Listening for tcp on: %s:%d", listener.localAddr, listener.port))
		}
		for {
			conn, acceptErr := netListener.Accept()
			if acceptErr != nil {
				fmt.Println(acceptErr)
			}

			args := &tcpArgs{conn: (&conn)}
			task := WorkerTask{
				args:       args,
				Execute:    executeTaskFunc,
				OnComplete: onCompleteFunc,
				OnError:    onErrorFunc,
			}

			select {
			case <-listener.ctx.Done():
				return
			default:

				if acceptErr == nil {
					workerpool.AddTask((&task))
				}
			}
		}
	}()
}

func (listener *TcpListener) CloseConnection()      { listener.cancelFunc() }
func (listener *TcpListener) ForceCloseConnection() { listener.cancelFunc() }

type tcpArgs struct {
	conn *net.Conn
}

func (t tcpArgs) isTask() bool {
	return true
}

func executeTaskFunc(w *Worker, task *WorkerTask) error {

	taskChannel := (*w).GetTaskChan()
	fmt.Printf("Worker Pointer: %p\n", w) // test to see if workers switch properly
	args := task.args.(*tcpArgs)
	conn := *(args.conn)
	conn.SetReadDeadline(time.Now().Add(time.Duration(w.taskTimeout) * time.Millisecond))
	buf := make([]byte, 4096)
	startIdx, err := conn.Read(buf)

	if err != nil {
		fmt.Println("read error: %v", err)
	} else {
		stringPayload := string(buf[:startIdx])
		//Strip off ending \n character for printing:
		fmt.Println(fmt.Sprintf("\n%s", stringPayload[:len(stringPayload)-1]))
	}
	if err != io.EOF {
		taskChannel <- task
	}
	return nil
}

func GetTcpListener(ctx context.Context, ipAddr string, port uint, workerCount uint) Listener {
	thisCtx, cancelFunc := context.WithCancel(ctx)

	baseListener := BaseListener{
		localAddr:     ipAddr,
		port:          port,
		outputChannel: make(chan NetMessage, 100),
		errorChannel:  make(chan error, 100),
		running:       false,
	}
	tcpListener := &TcpListener{
		BaseListener: baseListener,
		localAddr:    ipAddr,
		workerCount:  workerCount,
		ctx:          thisCtx,
		cancelFunc:   cancelFunc,
	}

	return tcpListener
}
