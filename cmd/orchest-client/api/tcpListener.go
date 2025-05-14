package api

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"
)

func getTcpListener(ctx context.Context, ipAddr string, port uint) (net.Listener, error) {
	listenerConfig := net.ListenConfig{
		KeepAliveConfig: net.KeepAliveConfig{Enable: true},
	}

	addr := fmt.Sprintf("%s:%d", ipAddr, port)

	listener, error := listenerConfig.Listen(ctx, TCP.String(), addr)
	return listener, error
}

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

			args := &tcpArgs{conn: (&conn)}
			task := WorkerTask{
				args: args,
				Execute: func(w *Worker, task *WorkerTask) error {
					output, err := executeTaskFunc(w, task)

					if err != nil {
						listener.errorChannel <- err
					}
					if len(output.GetData()) != 0 {
						listener.outputChannel <- output
					}
					return nil
				},
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

func executeTaskFunc(w *Worker, task *WorkerTask) (NetMessage, error) {
	taskChannel := (*w).GetTaskChan()
	//fmt.Printf("Worker Pointer: %p\n", w) // test to see if workers switch properly
	args := task.args.(*tcpArgs)
	conn := *(args.conn)
	conn.SetReadDeadline(time.Now().Add(time.Duration(w.taskTimeout) * time.Millisecond))
	buf := make([]byte, 4096)
	startIdx, err := conn.Read(buf)
	if err != io.EOF {
		taskChannel <- task
	}

	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		err = nil
	}
	output := NetMessage{
		data:      buf[:startIdx],
		protocol:  TCP,
		timestamp: time.Now(),
		origin:    conn.RemoteAddr().String(),
	}

	return output, err
}

func GetTcpListener(ctx context.Context, ipAddr string, port uint, workerCount uint, outputChan chan NetMessage, errChan chan error) Listener {
	thisCtx, cancelFunc := context.WithCancel(ctx)

	baseListener := BaseListener{
		localAddr:     ipAddr,
		port:          port,
		outputChannel: outputChan,
		errorChannel:  errChan,
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
