package api

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type NetProtocol string

const (
	TCP  NetProtocol = "tcp"
	UDP  NetProtocol = "udp"
	HTTP NetProtocol = "http"
)

func (p NetProtocol) String() string {
	return string(p)
}

type NetMessage struct {
	data      []byte
	protocol  NetProtocol
	timestamp time.Time
	origin    net.Addr
}

type Listener interface {
	getLocalAddr() net.Addr
	getChannel() chan NetMessage
	routine()
}

type HttpListener struct {
	channel   chan NetMessage
	localAddr net.Addr
}

func (listener HttpListener) getLocalAddr() net.Addr {
	return listener.localAddr
}

func getNetworkListener(ctx context.Context, ipAddr string, port uint, protocol NetProtocol) (net.Listener, error) {
	listenerConfig := net.ListenConfig{
		KeepAliveConfig: net.KeepAliveConfig{Enable: true},
	}

	addr := fmt.Sprintf("%s:%d", ipAddr, port)

	listener, error := listenerConfig.Listen(ctx, protocol.String(), addr)
	return listener, error
}

func httpRequestHandler(writer http.ResponseWriter, r *http.Request) {
	bytes, _ := io.ReadAll(r.Body)
	fmt.Println(string(bytes))

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	fmt.Fprintf(writer, `{"message":"orchest API is live!"}`)
}

func HttpApiRoutine(port uint, ctx context.Context) {
	mux := http.NewServeMux()

	mux.HandleFunc("/orchest/api", httpRequestHandler)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	fmt.Printf("Listening on http://%s/orchest/api\n", addr)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("HTTP server error:", err)
		}
	}()
	<-ctx.Done()
	fmt.Println("Closing HTTP API")
	server.Shutdown(context.Background())
}

func tcpConnectionHandler(ctx context.Context, conn *net.Conn) {
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
				go tcpConnectionHandler(ctx, &conn)
			}
		}
	}

}
