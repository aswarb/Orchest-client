package main

import (
	"context"
	"fmt"
	"net/http"
	"orchest-client/internal/api"
	"orchest-client/internal/config"
)

const (
	LOCALHOST = "127.0.0.1"
)

type HttpMethod string

const (
	GET     HttpMethod = "GET"
	HEAD    HttpMethod = "HEAD"
	POST    HttpMethod = "POST"
	PUT     HttpMethod = "PUT"
	DELETE  HttpMethod = "DELETE"
	CONNECT HttpMethod = "CONNECT"
	OPTIONS HttpMethod = "OPTIONS"
	TRACE   HttpMethod = "TRACE"
	PATCH   HttpMethod = "PATCH"
)

var (
	localEndpoints = []api.HttpEndpoint{
		api.MakeHTTPEndpoint("/api/services/",
			func(resWriter http.ResponseWriter, req *http.Request) {
				fmt.Println(req.Method)
				switch HttpMethod(req.Method) {
				case GET:
					resWriter.Header().Set("Content-Type", "application/json")
					fmt.Fprintf(resWriter, `{"data":"Hello world"}`)
					resWriter.WriteHeader(http.StatusOK)
				default:
					resWriter.WriteHeader(http.StatusMethodNotAllowed)
				}
			}),
	}
)

func GetLocalhostAPI(ctx context.Context, outputChannel chan api.NetMessage, errorChannel chan error) (*api.HttpListener, error) {
	conf, err := config.GetGatewayConfig()
	fmt.Println(conf.Gateway)
	if err != nil {
		return nil, err
	}
	port := uint(conf.Gateway.Port)

	listener := api.GetHttpListener(ctx, LOCALHOST, uint(port), localEndpoints, outputChannel, errorChannel)
	return listener, nil
}

func GetNetworkAPI() (*api.HttpListener, error) {

	return nil, nil
}
