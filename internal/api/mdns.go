package api

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type IPType int

// could use iota, using 4 and 6 here is clearer though
const (
	IPv4 IPType = 4
	IPv6 IPType = 6
)

type MdnsResult struct {
	Iface    *net.Interface
	Instance string
	Service  string
	Domain   string
	Hostname string
	IP       net.IP
	Port     int
	IPType   IPType
	Txt      []string
}

type MdnsQuery interface {
	Start() (chan *MdnsResult, error)
	Stop()
	GetResultsChan() <-chan *MdnsResult
}

type AvahiQuery struct {
	instanceSubstr string
	serviceSubstr  string
	domainSubstr   string
	hostnameSubstr string

	stopChan    chan struct{}
	resultsChan chan *MdnsResult

	isRunning bool
	mu        sync.Mutex
}

func (q *AvahiQuery) Start() (chan *MdnsResult, error) {
	q.mu.Lock()
	if q.isRunning {
		q.mu.Unlock()
		return q.resultsChan, nil
	}
	// Remake channel so it can be used to signal data is finished for synchronous fetching of data from this async function
	q.resultsChan = make(chan *MdnsResult, 10)
	q.mu.Unlock()
	execStr := "avahi-browse"
	args := []string{"-arpt"}
	queryCmd := exec.Command(execStr, args...)
	reader, err := queryCmd.StdoutPipe()

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	parseStdout := func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()

			start := string(line[0])
			if start != "=" {
				continue
			}
			lineParts := strings.Split(line, ";")

			ifacestr := lineParts[1]
			iptypestr := lineParts[2]
			instanceName := lineParts[3]
			service := lineParts[4]
			domain := lineParts[5]
			hostname := lineParts[6]
			address := lineParts[7]
			portstr := lineParts[8]
			txt := lineParts[9:]

			if q.instanceSubstr != "" && !strings.Contains(instanceName, q.instanceSubstr) {
				continue
			}
			if q.serviceSubstr != "" && !strings.Contains(service, q.serviceSubstr) {
				continue
			}
			if q.domainSubstr != "" && !strings.Contains(domain, q.domainSubstr) {
				continue
			}
			if q.hostnameSubstr != "" && !strings.Contains(hostname, q.hostnameSubstr) {
				continue
			}

			iface, err := net.InterfaceByName(ifacestr)
			if err != nil {
				fmt.Println(err)
				continue
			}

			var addressType IPType
			switch iptypestr {
			case "IPv4":
				addressType = IPv4
			case "IPv6":
				addressType = IPv6
			}

			port, err := strconv.ParseInt(portstr, 10, 32)
			if err != nil {
				fmt.Println(err)
				continue
			}
			entry := MakeMdnsResult(iface, instanceName, service, domain, hostname, int(port), net.ParseIP(address), addressType, txt)
			fmt.Println(entry)
			q.resultsChan <- entry
		}
		fmt.Println("Scanner finished")
		q.Stop()
	}

	waitForStopSignal := func() {
		<-q.stopChan
		queryCmd.Process.Kill()

		q.mu.Lock()
		close(q.resultsChan)
		q.isRunning = false
		q.mu.Unlock()
	}

	startErr := queryCmd.Start()

	if startErr != nil {
		return nil, startErr
	}

	q.mu.Lock()
	q.isRunning = true
	q.mu.Unlock()

	go waitForStopSignal()
	go parseStdout()

	return q.resultsChan, nil
}

func (q *AvahiQuery) Stop() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.isRunning {
		q.stopChan <- struct{}{}
	}
}

func (q *AvahiQuery) GetResultsChan() (<-chan *MdnsResult, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.isRunning {
		return q.resultsChan, nil
	} else {
		return nil, fmt.Errorf("Channel is not open. Start query to create an open channel")
	}
}

func MakeAvahiQuery(instanceSubstr string, serviceSubstr string, domainSubstr string, hostnameSubstr string) *AvahiQuery {
	outChan := make(chan *MdnsResult, 10)

	q := AvahiQuery{
		instanceSubstr: instanceSubstr,
		serviceSubstr:  serviceSubstr,
		domainSubstr:   domainSubstr,
		hostnameSubstr: hostnameSubstr,
		resultsChan:    outChan,
		stopChan:       make(chan struct{}),
	}

	return &q
}

func MakeMdnsResult(Iface *net.Interface, Instance string, Service string, Domain string, Hostname string, Port int, IP net.IP, IPType IPType, Txt []string) *MdnsResult {
	r := MdnsResult{
		Iface:    Iface,
		Instance: Instance,
		Service:  Service,
		Domain:   Domain,
		Hostname: Hostname,
		Port:     Port,
		IP:       IP,
		IPType:   IPType,
		Txt:      Txt,
	}
	return &r
}
