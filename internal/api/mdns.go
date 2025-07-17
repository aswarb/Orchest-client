package api

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/hashicorp/mdns"
)

func StartMdnsDiscover(query *mdns.QueryParam) error {
	var err error
	err = mdns.Query(query)
	return err
}

func StopMdnsDiscover(query *mdns.QueryParam) {
	close(query.Entries)
}

func StartMdnsBroadcast(instance string, service string, domain string, hostName string, port int, ips []net.IP, txt []string) (func(), error) {
	mdnsService, e1 := mdns.NewMDNSService(instance, service, domain, hostName, port, ips, txt)
	fmt.Println(mdnsService)
	config := mdns.Config{Zone: mdnsService, Iface: nil, LogEmptyResponses: false, Logger: nil}
	Server, e2 := mdns.NewServer(&config)

	ctx, cancelFunc := context.WithCancel(context.Background())

	serverKillFunc := func() {
		<-ctx.Done()
		Server.Shutdown()
	}
	go serverKillFunc()

	return cancelFunc, errors.Join(e1, e2)
}
