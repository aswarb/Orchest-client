package main

import (
	"context"
	"fmt"
	"net"
	"orchest-client/internal/api"
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

	httpListener, getListenerErr := GetLocalhostAPI(ctx, outputChan, errorChannel)
	if getListenerErr == nil {
		fmt.Println("Trying to start http listener")
		httpListener.OpenConnection()
	}
	tcpListener := api.GetTcpListener(ctx, ipParts[0], 1025, 5, outputChan, errorChannel)
	tcpListener.OpenConnection()

	udpListener := api.GetUdpListener(ctx, ipParts[0], 1026, 5, outputChan, errorChannel)
	udpListener.OpenConnection()

	mdnsQuery := api.MakeAvahiQuery("", "_orchest", "", "")

	mdnsQuery.Start()

	host, _ := os.Hostname()
	fmt.Println("Starting mdns broastcast on ", host)
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
	<-sigChan
	mdnsQuery.Stop()
	fmt.Println("Shutdown signal received. Cleaning up...")
	cancelFunc()

}
