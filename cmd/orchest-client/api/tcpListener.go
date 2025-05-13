package api

import (
	"context"
	"fmt"
	"net"
	"time"
	"io"
"sync"
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
	localAddr string
	workerCount uint
}

func connectionHandler(ctx context.Context, conn *net.Conn) {
	var arr []string
	defer (*conn).Close()

	for {
		buf := make([]byte, 4096)
		startIdx, err := (*conn).Read(buf)

		if err != nil {
			fmt.Println("read error: %v", err)
			break
		}
		stringPayload := string(buf[:startIdx])
		arr = append(arr, stringPayload)
		//Strip off ending \n character for printing:
		fmt.Println(fmt.Sprintf("\n%s", stringPayload[:len(stringPayload)-1]))
	}
}

type tcpArgs struct {
	conn *net.Conn
}

func (t tcpArgs) isTask() bool {
	return true
}

func (listener *TcpListener) OpenConnection(ctx context.Context) {

	workerpool := MakeWorkerPool(ctx)
	workerpool.AddWorkers(listener.workerCount)

	go func() {
		netListener, setupErr := getTcpListener(ctx, listener.localAddr, listener.port)

		workerpool.StartWork(ctx)
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
			taskChannel := workerpool.GetTaskChan()

			executeTaskFunc := func(w *Worker, task *WorkerTask) error {
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
				taskChannel <- task}
				return nil
			}

			OnCompleteFunc := func(w *Worker, task *WorkerTask) {}
			OnErrorFunc := func(w *Worker, task *WorkerTask, e error) {}

			args := &tcpArgs{conn: (&conn)}
			task := WorkerTask{
				args:       args,
				Execute:    executeTaskFunc,
				OnComplete: OnCompleteFunc,
				OnError:    OnErrorFunc,
			}

			select {
			case <-ctx.Done():
				return
			default:

				if acceptErr == nil {
					workerpool.AddTask((&task))
				}
			}
		}
	}()
}

func (listener *TcpListener) CloseConnection() {
}
func (listener *TcpListener) ForceCloseConnection() {
}

func GetTcpListener(ipAddr string, port uint, workerCount uint) Listener {
	//addr := fmt.Sprintf("%s:%d", ipAddr, port)
	var wg sync.WaitGroup
	baseListener := BaseListener{
		localAddr:     ipAddr,
		port:          port,
		outputChannel: make(chan NetMessage, 100),
		errorChannel:  make(chan error, 100),
		wg:            wg,
		running:       false,
	}
	tcpListener := &TcpListener{
		BaseListener: baseListener,
		localAddr:ipAddr,
		workerCount: workerCount,
	}

	return tcpListener
}
