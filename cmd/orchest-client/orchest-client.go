package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"orchest-client/internal/api"
	"orchest-client/internal/task"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Request interface {
	getReplyChan() chan Response
}

type PushRequest struct {
	reply   chan Response
	payload Command
}

func (p PushRequest) getReplyChan() chan Response {
	return p.reply
}

type PopRequest struct {
	reply chan Response
	wait  bool
}

func (p PopRequest) getReplyChan() chan Response {
	return p.reply
}

type Response struct {
	payload Command
	ok      bool
}

func listener(sleeptime int64, requestChannel chan<- Request, ctx context.Context) {

	fmt.Println("Listener Started")
	replyChannel := make(chan Response)

	for i := range 100 {

		t := START_PROCESS_MESSAGE
		if i%2 == 1 {
			t = STOP_PROCESS_MESSAGE
		}
		cmd := Command{messageType: t, payload: strconv.Itoa(i)}

		requestChannel <- PushRequest{
			reply:   replyChannel,
			payload: cmd,
		}
		time.Sleep(time.Duration(sleeptime) * time.Millisecond)
	}

}

func consumer(sleeptime int64, requestChannel chan<- Request, ctx context.Context) {
	replyChannel := make(chan Response)
	requestChannel <- PopRequest{
		reply: replyChannel,
	}
	for rep := range replyChannel {

		requestChannel <- PopRequest{
			reply: replyChannel,
		}
		rep.payload.Execute()
		fmt.Print("\r")

	}
}

func queueManager(sleeptime int64, requestChannel <-chan Request, ctx context.Context) {

	var popQueue []PopRequest
	var pushQueue []PushRequest

	fmt.Println("Queue manager started")
	for req := range requestChannel {

		switch r := req.(type) {
		case PushRequest:
			pushQueue = append(pushQueue, r)

		case PopRequest:
			popQueue = append(popQueue, r)
		}
		for len(popQueue) != 0 {
			if len(pushQueue) == 0 {
				break
			} else {
				request := popQueue[0]
				payload := pushQueue[0].payload

				request.reply <- Response{payload: payload, ok: true}

				popQueue = popQueue[1:]
				pushQueue = pushQueue[1:]

			}
		}
		time.Sleep(time.Duration(sleeptime) * time.Millisecond)
	}
}

func main() {
	requestChannel := make(chan Request)
	ctx, cancelFunc := context.WithCancel(context.Background())

	interfaces, _ := net.Interfaces()
	var validIfaces []net.Interface
	// Get a valid network interface - needs flags listed here, exact IP doesn't matter
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagBroadcast == 0 {
			continue
		}
		if iface.Flags&net.FlagRunning == 0 {
			continue
		}
		if iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		validIfaces = append(validIfaces, iface)
	}

	defualtHandleEndpoint := func(w http.ResponseWriter, r *http.Request) (bool, []byte) {
		bytes, e := io.ReadAll(r.Body)
		fmt.Println(string(bytes))
		fmt.Println(e)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message":"orchest API is live!"}`)
		return true, bytes
	}

	httpEndpoints := [...]api.HttpEndpoint{
		api.HttpEndpoint{
			RelativePath: "/orchest/api",
			Handler:      defualtHandleEndpoint,
		},
	}

	addrs, _ := validIfaces[0].Addrs()
	ipParts := strings.Split(addrs[0].String(), "/")

	go listener(100, requestChannel, ctx)
	go queueManager(100, requestChannel, ctx)
	go consumer(100, requestChannel, ctx)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Program is running. Press Ctrl+C to exit.")

	errorChannel := make(chan error, 1000)
	outputChan := make(chan api.NetMessage, 1000)

	httpListener := api.GetHttpListener(ctx, "127.0.0.1", 1024, httpEndpoints[:], outputChan, errorChannel)
	httpListener.OpenConnection()

	tcpListener := api.GetTcpListener(ctx, ipParts[0], 1025, 5, outputChan, errorChannel)
	tcpListener.OpenConnection()

	udpListener := api.GetUdpListener(ctx, ipParts[0], 1026, 5, outputChan, errorChannel)
	udpListener.OpenConnection()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errorChannel:
				fmt.Print("\n")
				fmt.Println(err)
			case out := <-outputChan:
				fmt.Print("\n")
				fmt.Println(out.GetDataString())
			}
		}
	}()
	wd, _ := os.Getwd()
	tomlPath := fmt.Sprintf("%s%s", wd, "/internal/task/TEMPLATE_test6.orchest.task.toml")
	fmt.Println(tomlPath)
	tasks := task.GetTomlTaskArray(tomlPath)
	fmt.Println("TomlTasks:", tasks)
	for _, task := range tasks {
		fmt.Println(task)
	}
	taskManager := task.GetTaskManagerFromToml(tasks)
	fmt.Println(taskManager)
	taskManager.StartTask(ctx)

	<-sigChan
	fmt.Println("Shutdown signal received. Cleaning up...")
	cancelFunc()

}
