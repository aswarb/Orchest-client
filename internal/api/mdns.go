package api

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"sync"
)

type MdnsResult struct {
	Instance string
	Service  string
	Domain   string
	Hostname string
	Port     int
	IPs      []net.IP
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
			fmt.Println(scanner.Text())
		}
		fmt.Println("Scanner finished")
		q.Stop()
	}

	waitForStopSignal := func() {
		<-q.stopChan
		queryCmd.Process.Kill()

		q.mu.Lock()
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

func (q *AvahiQuery) GetResultsChan() <-chan *MdnsResult {
	return q.resultsChan
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

func MakeMdnsResult(Instance string, Service string, Domain string, Hostname string, Port int, IPs []net.IP, Txt []string) *MdnsResult {
	r := MdnsResult{
		Instance: Instance,
		Service:  Service,
		Domain:   Domain,
		Hostname: Hostname,
		Port:     Port,
		IPs:      IPs,
		Txt:      Txt,
	}
	return &r
}
