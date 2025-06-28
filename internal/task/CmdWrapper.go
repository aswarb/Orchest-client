package task

import (
	"context"
	"errors"
	"io"
	wp "orchest-client/internal/workerpool"
	"os/exec"
	"sync"
)

type PipeWrapper struct {
	mu     sync.Mutex
	closed bool // should reflect whether or not .Close() has been invoked on pipeWrapper.in and PipeWrapper.out, not a toggle
	in     io.WriteCloser
	out    io.ReadCloser
}

func (p *PipeWrapper) GetInWriter() io.WriteCloser { return p.in }
func (p *PipeWrapper) GetOutReader() io.ReadCloser { return p.out }

func (p *PipeWrapper) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	e1 := p.in.Close()
	e2 := p.out.Close()

	err := errors.Join(e1, e2)
	if err != nil {
		return err
	}
	p.closed = false
	return nil
}

func (p *PipeWrapper) Write(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, err := p.in.Write(data)
	return err
}

func (p *PipeWrapper) Read(target []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	n, err := p.out.Read(target)
	return n, err
}
func (p *PipeWrapper) IsOpen() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	val := p.closed
	return val
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
