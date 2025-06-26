package task

import (
	"errors"
	"io"
	"os/exec"
	"sync"
)

type PipeEndpoint interface {
	isOpen() bool
	Close() error
}

type OutPipeEndpoint struct {
	mu     sync.Mutex
	closed bool // should reflect whether or not .Close() has been invoked on InPipeEndpoint.end, not a toggle
	end    io.ReadCloser
}

func (p *OutPipeEndpoint) Close() error {
	err := p.end.Close()
	return err
}

func (p *OutPipeEndpoint) isOpen() bool {
	return p.closed
}

func (p *OutPipeEndpoint) GetEndpoint() io.ReadCloser {
	return p.end
}

type InPipeEndpoint struct {
	mu     sync.Mutex
	closed bool // should reflect whether or not .Close() has been invoked on InPipeEndpoint.end, not a toggle
	end    io.WriteCloser
}

func (p *InPipeEndpoint) Close() error {
	err := p.end.Close()
	return err
}

func (p *InPipeEndpoint) isOpen() bool {
	return p.closed
}
func (p *InPipeEndpoint) GetEndpoint() io.WriteCloser {
	return p.end
}

func MakeInPipeEndpoint(end io.WriteCloser, closed bool) *InPipeEndpoint {
	endpoint := &InPipeEndpoint{
		mu:     sync.Mutex{},
		closed: closed,
		end:    end,
	}
	return endpoint
}

func MakeOutPipeEndpoint(end io.ReadCloser, closed bool) *OutPipeEndpoint {
	endpoint := &OutPipeEndpoint{
		mu:     sync.Mutex{},
		closed: closed,
		end:    end,
	}
	return endpoint
}

type ProcessEndpoint struct {
	cmd    *exec.Cmd
	in     *OutPipeEndpoint
	out    *InPipeEndpoint
	buffer Buffer
}

func (p *ProcessEndpoint) BufferedStdinWrite(data []byte) {

}

func (p *ProcessEndpoint) ExecuteBlocking() error {
	err := p.cmd.Run()
	return err
}

func (p *ProcessEndpoint) SetInPipeEndpoint(endpoint *OutPipeEndpoint) {
	p.in = endpoint
	p.cmd.Stdin = endpoint.GetEndpoint()
}

func (p *ProcessEndpoint) SetOutPipeEndpoint(endpoint *InPipeEndpoint) {
	p.out = endpoint
	p.cmd.Stdout = endpoint.GetEndpoint()
}

func (p *ProcessEndpoint) ClosePipes() error {
	e1 := p.in.Close()
	e2 := p.out.Close()

	return errors.Join(e1, e2)
}
