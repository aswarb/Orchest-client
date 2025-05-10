package api

import (
	"context"
	"sync"
	"time"
)

type NetProtocol string

const (
	HTTP NetProtocol = "http"
	TCP  NetProtocol = "tcp"
	UDP  NetProtocol = "udp"
)

func (protocol NetProtocol) String() string {
	return string(protocol)
}

type NetMessage struct {
	data      []byte
	protocol  NetProtocol
	timestamp time.Time
	origin    string
}

type Listener interface {
	GetLocalAddr() string
	GetChannel() chan NetMessage
	GetWaitGroup() sync.WaitGroup
	OpenConnection(context.Context)
	CloseConnection()
	ForceCloseConnection()
}

type BaseListener struct {
	localAddr     string
	port          uint
	outputChannel chan NetMessage
	errorChannel  chan error
	wg            sync.WaitGroup
	running       bool
}

func (listener *BaseListener) GetLocalAddr() string         { return listener.localAddr }
func (listener *BaseListener) GetChannel() chan NetMessage  { return listener.outputChannel }
func (listener *BaseListener) GetWaitGroup() sync.WaitGroup { return listener.wg }
func (listener *BaseListener) IsRunning() bool              { return listener.running }
