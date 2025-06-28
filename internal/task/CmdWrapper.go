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
	cmd            *exec.Cmd
	inPipe         *PipeWrapper
	outPipe        *PipeWrapper
	fromBufferPipe *PipeWrapper
	buffer         Buffer
}

func (p *ProcessEndpoint) BufferedStdinWrite(data []byte) {

}

func (p *ProcessEndpoint) ExecuteBlocking() error {
	err := p.cmd.Run()
	return err
}

func (p *ProcessEndpoint) EnableBuffer() {
	p.cmd.Stdin = p.fromBufferPipe.GetOutReader()
}

type consumerTaskArgs struct {
}

func (c *consumerTaskArgs) IsTask() bool

func (p *ProcessEndpoint) startStdinConsumers(ctx context.Context) {
	consumerFuncExecute := func(w *wp.Worker, wt *wp.WorkerTask) error {
		for {
			data := make([]byte, 4096)
			n, err := p.inPipe.Read(data)
			data = data[:n]
			if err == io.EOF {
				return io.EOF
			}

			p.buffer.Write(data)
		}
	}

	consumerFuncComplete := func(*wp.Worker, *wp.WorkerTask) {}
	consumerFuncError := func(*wp.Worker, *wp.WorkerTask, error) {}

	workerpool := wp.MakeWorkerPool(ctx)
	taskArgs := consumerTaskArgs{}
	workerpool.AddWorkers(uint(1))
	task := wp.WorkerTask{Args: &taskArgs,
		Execute:    consumerFuncExecute,
		OnComplete: consumerFuncComplete,
		OnError:    consumerFuncError,
	}
	workerpool.AddTask(&task)
	workerpool.StartWork(ctx)
}

func (p *ProcessEndpoint) DisableBuffer() {

}

func (p *ProcessEndpoint) ClosePipes() error {
	e1 := p.inPipe.Close()
	e2 := p.outPipe.Close()

	return errors.Join(e1, e2)
}
