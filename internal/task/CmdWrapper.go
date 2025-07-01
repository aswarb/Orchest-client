package task

import (
	"context"
	"errors"
	"fmt"
	"io"
	wp "orchest-client/internal/workerpool"
	"os/exec"
)

type CmdWrapper struct {
	cmd             *exec.Cmd
	inPoint         io.ReadCloser  // The point the cmd will read from to get stdin input
	outPoint        io.WriteCloser // The piont the cmd will write to to give stdout output
	bufPipeInPoint  io.WriteCloser // "in" point for the new pipe for buffer -> cmd.stdin communication
	bufPipeOutPoint io.ReadCloser  // "out" point for the new pipe. this is the end assigned to cmd.Stdin for the buffer to write to
	buffer          Buffer
	bufferEnabled   bool
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
	p.cmd.Stdin = p.inPoint
}

func (p *CmdWrapper) ClosePipes() error {
	e1 := p.CloseStdin()
	e2 := p.CloseStdout()
	e3 := p.CloseBufStdin()
	e4 := p.CloseStdout()

	return errors.Join(e1, e2, e3, e4)
}

func (p *CmdWrapper) CloseStdin() error {
	e1 := p.inPoint.Close()

	return e1
}
func (p *CmdWrapper) CloseStdout() error {
	e2 := p.outPoint.Close()

	return e2
}

func (p *CmdWrapper) CloseBufStdin() error {
	e3 := p.bufPipeInPoint.Close()
	return e3
}
func (p *CmdWrapper) CloseBufStdout() error {
	e4 := p.bufPipeOutPoint.Close()
	return e4
}
func CreateCmdWrapper(executable string, args []string, inPoint io.ReadCloser, outPoint io.WriteCloser) *CmdWrapper {
	bufReader, bufWriter := io.Pipe()

	cmd := exec.Command(executable, args...)
	buffer := MakeExtensibleBuffer(5)

	wrapper := CmdWrapper{cmd: cmd,
		inPoint:         inPoint,
		outPoint:        outPoint,
		bufPipeInPoint:  bufWriter,
		bufPipeOutPoint: bufReader,
		buffer:          buffer,
		bufferEnabled:   false,
	}

	return &wrapper
}
