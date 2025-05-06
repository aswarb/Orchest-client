package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
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

func listener(sleeptime int64, requestChannel chan<- Request) {

	fmt.Println("Listener Started")
	replyChannel := make(chan Response)

	//var arr []Command
	for i := 0; i < 100; i++ {

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

func consumer(sleeptime int64, requestChannel chan<- Request) {
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

func queueManager(sleeptime int64, requestChannel <-chan Request) {

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

	go listener(100, requestChannel)
	go queueManager(100, requestChannel)
	go consumer(100, requestChannel)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Program is running. Press Ctrl+C to exit.")
	<-sigChan

	fmt.Println("Shutdown signal received. Cleaning up...")

}
