package api

import (
	"context"
	"fmt"
	"net/http"
)

type HandlerFunc func(http.ResponseWriter, *http.Request)

type HttpListener struct {
	BaseListener
	server     *http.Server
	mux        *http.ServeMux
	endpoints  []HttpEndpoint
	ctx        context.Context
	cancelFunc func()
}

func (listener *HttpListener) OpenConnection() {
	for _, endpoint := range listener.endpoints {
		handleFunc := func(writer http.ResponseWriter, request *http.Request) {
			fmt.Println(request)
			endpoint.Handler(writer, request)
		}
		listener.mux.HandleFunc(endpoint.RelativePath, handleFunc)
		fmt.Printf("Listening on http://%s:%d%s\n", listener.localAddr, listener.port, endpoint.RelativePath)
	}

	go func() {
		if err := listener.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("HTTP server error:", err)
		}
	}()
	go func() {
		<-listener.ctx.Done()
		listener.CloseConnection()
	}()
}
func (listener *HttpListener) CloseConnection() {
	fmt.Println("Closing HTTP API")
	listener.server.Shutdown(listener.ctx)
}
func (listener *HttpListener) ForceCloseConnection() {
	listener.server.Close()
}

type HttpEndpoint struct {
	RelativePath string
	Handler      HandlerFunc
}

func (h HttpEndpoint) PathMatches(path string) bool {
	return h.RelativePath == path
}

func MakeHTTPEndpoint(relativePath string, handlerFunc HandlerFunc) HttpEndpoint {
	return HttpEndpoint{RelativePath: relativePath, Handler: handlerFunc}
}

func GetHttpListener(ctx context.Context, ipAddr string, port uint, endpoints []HttpEndpoint, outputChan chan NetMessage, errChan chan error) *HttpListener {
	thisCtx, cancelFunc := context.WithCancel(ctx)
	mux := http.NewServeMux()

	addr := fmt.Sprintf("%s:%d", ipAddr, port)
	server := http.Server{
		Addr:    addr,
		Handler: mux,
	}
	baseListener := BaseListener{localAddr: ipAddr,
		port:          port,
		outputChannel: outputChan,
		errorChannel:  errChan,
		running:       false,
	}
	httpListener := &HttpListener{
		BaseListener: baseListener,
		server:       &server,
		endpoints:    endpoints,
		mux:          mux,
		ctx:          thisCtx,
		cancelFunc:   cancelFunc,
	}

	return httpListener
}
