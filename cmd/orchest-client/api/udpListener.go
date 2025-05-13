package api

import (
	"context"
	"fmt"
	"net"
)

type UdpListener struct {
	channel   chan NetMessage
	localAddr net.Addr
}

func (listener *UdpListener) openConnection() {
}
func (listener *UdpListener) closeConnection() {
}
func (listener *UdpListener) forceCloseConnection() {
}

func getNetworkListener(ctx context.Context, ipAddr string, port uint, protocol NetProtocol) (net.Listener, error) {

	listenerConfig := net.ListenConfig{
		KeepAliveConfig: net.KeepAliveConfig{Enable: true},
	}

	addr := fmt.Sprintf("%s:%d", ipAddr, port)

	listener, error := listenerConfig.Listen(ctx, TCP.String(), addr)
	return listener, error

}

func UdpListenerRoutine(ctx context.Context, ipAddr string, port uint) {

	listener, setupErr := getNetworkListener(ctx, ipAddr, port, UDP)
	if setupErr != nil {
		fmt.Println(setupErr)
	} else {
		fmt.Println(fmt.Sprintf("Listening for tcp on: %s:%d", ipAddr, port))
	}
	for {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			fmt.Println(acceptErr)
		}
		select {
		case <-ctx.Done():
			return
		default:
			if acceptErr == nil {
				go connectionHandler(ctx, &conn)
			}
		}
	}

}
func GetUdpListener() *Listener {
	return nil
}
