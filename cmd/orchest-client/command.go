package main

import (
	"fmt"
)

type MessageType int

const (
	START_PROCESS_MESSAGE MessageType = iota
	STOP_PROCESS_MESSAGE
)

type Command struct {
	messageType MessageType
	payload     string
}

func (c *Command) Execute() error {
	//placeholder
	fmt.Println("Handling Command of type: ", c.messageType)
	switch c.messageType {
	case START_PROCESS_MESSAGE:
		return startProcessMsg(c.payload)
	case STOP_PROCESS_MESSAGE:
		return stopProcessMsg(c.payload)
	default:
		fmt.Println("Message of messageType ", c.messageType, "not handled, unknown messageType")
		return nil
	}
}

func startProcessMsg(payload string) error {
	fmt.Println(payload)
	return nil
}

func stopProcessMsg(payload string) error {
	fmt.Println(payload)
	return nil
}
