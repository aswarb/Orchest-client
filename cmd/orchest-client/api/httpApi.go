package api

import (
	"fmt"
	"net/http"
	"net"
	"time"
)

type NetProtocol string

const (
	TCP  NetProtocol = "tcp"
	UDP  NetProtocol = "udp"
	HTTP NetProtocol = "http"
)

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

func httpRequestHandler(writer http.ResponseWriter, r *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	fmt.Fprintf(writer, `{"message":"orchest API is live!"}`)
}

func StartHttpApi(port uint) {
	mux := http.NewServeMux()

	mux.HandleFunc("/Orchest/api", httpRequestHandler)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	fmt.Println(addr)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	
	server.ListenAndServe()
}
