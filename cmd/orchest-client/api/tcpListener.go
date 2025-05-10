package api

import (
	"context"
	"fmt"
	"net"
)

func getNetworkListener(ctx context.Context, ipAddr string, port uint, protocol NetProtocol) (net.Listener, error) {
	listenerConfig := net.ListenConfig{
		KeepAliveConfig: net.KeepAliveConfig{Enable: true},
	}

	addr := fmt.Sprintf("%s:%d", ipAddr, port)

	listener, error := listenerConfig.Listen(ctx, protocol.String(), addr)
	return listener, error
}

type TcpListener struct {
	channel   chan NetMessage
	localAddr net.Addr
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

func TcpListenerRoutine(ctx context.Context, ipAddr string, port uint) {

	listener, setupErr := getNetworkListener(ctx, ipAddr, port, TCP)
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

func (listener *TcpListener) openConnection(ctx context.Context) {
}
func (listener *TcpListener) closeConnection() {
}
func (listener *TcpListener) forceCloseConnection() {
}

func GetTcpListener() *Listener {
	return nil
}
