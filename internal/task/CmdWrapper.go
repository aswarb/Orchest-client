package task

import (
	"context"
	"errors"
	"io"
	wp "orchest-client/internal/workerpool"
	"os/exec"
)

type CmdWrapper struct {
	cmd            *exec.Cmd
	inPipe         *PipeWrapper
	outPipe        *PipeWrapper
	fromBufferPipe *PipeWrapper
	buffer         Buffer
}

func (p *CmdWrapper) BufferedStdinWrite(data []byte) {

}

func (p *CmdWrapper) ExecuteBlocking() error {
	err := p.cmd.Run()
	return err
}

func (p *CmdWrapper) EnableBuffer() {
	p.cmd.Stdin = p.fromBufferPipe.GetOutReader()
}

type consumerTaskArgs struct {
}

func (c *consumerTaskArgs) IsTask() bool

func (p *CmdWrapper) startStdinConsumers(ctx context.Context) {
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

func (p *CmdWrapper) DisableBuffer() {

}

func (p *CmdWrapper) ClosePipes() error {
	e1 := p.inPipe.Close()
	e2 := p.outPipe.Close()
	e3 := p.fromBufferPipe.Close()

	return errors.Join(e1, e2, e3)
}
