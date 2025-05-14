package api

import (
	"context"
	"fmt"
	"net"
	"time"
)

type UdpListener struct {
	BaseListener
	workerCount uint
	ctx         context.Context
	cancelFunc  func()
}

func (listener *UdpListener) OpenConnection() {

	go func() {
		addr := net.UDPAddr{IP: net.ParseIP(listener.GetLocalAddr()), Port: int(listener.port)}
		fmt.Println(fmt.Sprintf("Listening for udp on: %s:%d", listener.GetLocalAddr(), listener.port))
		conn, acceptErr := net.ListenUDP("udp", &addr)
		if acceptErr != nil {
			listener.errorChannel <- acceptErr
		}
		for {
			select {
			case <-listener.ctx.Done():
				return
			default:
				if acceptErr == nil {
					buf := make([]byte, 4096)
					num, addr, err := conn.ReadFromUDP(buf)
					if err != nil {
						listener.errorChannel <- err
					}
					output := NetMessage{
						data:      buf[:num],
						protocol:  UDP,
						timestamp: time.Now(),
						origin:    addr.String(),
					}
					if len(output.GetData()) != 0 {
						listener.outputChannel <- output
					}
				}
			}
		}
	}()
}

func (listener *UdpListener) CloseConnection()      { listener.cancelFunc() }
func (listener *UdpListener) ForceCloseConnection() { listener.cancelFunc() }

func GetUdpListener(ctx context.Context, ipAddr string, port uint, workerCount uint, outputChan chan NetMessage, errChan chan error) Listener {
	thisCtx, cancelFunc := context.WithCancel(ctx)
	baseListener := BaseListener{
		localAddr:     ipAddr,
		port:          port,
		outputChannel: outputChan,
		errorChannel:  errChan,
		running:       false,
	}
	tcpListener := &UdpListener{
		BaseListener: baseListener,
		workerCount:  workerCount,
		ctx:          thisCtx,
		cancelFunc:   cancelFunc,
	}

	return tcpListener
}
