package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type HttpListener struct {
	BaseListener
	server    *http.Server
	mux       *http.ServeMux
	endpoints []HttpEndpoint
}

func (listener *HttpListener) OpenConnection(ctx context.Context) {
	for _, endpoint := range listener.endpoints {

		handleFunc := func(writer http.ResponseWriter, request *http.Request) {
			fmt.Println(request)
			shouldOutput, outputvalue := endpoint.Handler(writer, request)
			if shouldOutput {
				listener.outputChannel <- NetMessage{
					data:      outputvalue,
					protocol:  HTTP,
					timestamp: time.Now(),
					origin:    request.RemoteAddr,
				}
			}
		}
		listener.mux.HandleFunc(fmt.Sprintf("%s", endpoint.RelativePath), handleFunc)
		fmt.Printf("Listening on http://%s:%d%s\n", listener.localAddr, listener.port, endpoint.RelativePath)

	}

	go func() {
		if err := listener.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("HTTP server error:", err)
		}
	}()
	go func() {
		<-ctx.Done()
		listener.CloseConnection()
	}()
}
func (listener *HttpListener) CloseConnection() {
	fmt.Println("Closing HTTP API")
	listener.server.Shutdown(context.Background())
}
func (listener *HttpListener) ForceCloseConnection() {
	listener.server.Close()
}

type HttpEndpoint struct {
	RelativePath string
	Handler      func(http.ResponseWriter, *http.Request) (bool, []byte)
}

func GetHttpListener(ipAddr string, port uint, endpoints []HttpEndpoint) Listener {
	mux := http.NewServeMux()

	addr := fmt.Sprintf("%s:%d", ipAddr, port)
	server := http.Server{
		Addr:    addr,
		Handler: mux,
	}
	var wg sync.WaitGroup
	baseListener := BaseListener{
		localAddr:     ipAddr,
		port:          port,
		outputChannel: make(chan NetMessage, 100),
		errorChannel:  make(chan error, 100),
		wg:            wg,
		running:       false,
	}
	httpListener := &HttpListener{
		BaseListener: baseListener,
		server:       &server,
		endpoints:    endpoints,
		mux:          mux,
	}

	return httpListener
}
