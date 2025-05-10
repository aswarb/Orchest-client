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

func (listener *UdpListener) openConnection(ctx context.Context) {
}
func (listener *UdpListener) closeConnection() {
}
func (listener *UdpListener) forceCloseConnection() {
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
