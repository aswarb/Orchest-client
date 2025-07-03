package task

import (
	"context"
	"errors"
	"fmt"
	"io"
	wp "orchest-client/internal/workerpool"
	"os"
	"os/exec"
)

type DummyCloser struct {
}

func (d *DummyCloser) Close() error                { return nil }
func (d *DummyCloser) Read(p []byte) (int, error)  { return 0, nil }
func (d *DummyCloser) Write(p []byte) (int, error) { return 0, nil }

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
	p.cmd.Stderr = os.Stdout
	if p.cmd.Stdout == nil {
		p.cmd.Stdout = os.Stdout
	}
	err := p.cmd.Run()
	return err
}

func (p *CmdWrapper) EnableBuffer(ctx context.Context) {
	if !p.bufferEnabled {
		p.cmd.Stdin = p.bufPipeOutPoint

		p.startStdinConsumer(ctx) // consumer for stdin point -> buffer
		p.startBufConsumer(ctx)   // consumer for buffer -> true cmd stdin
		p.bufferEnabled = true
	}
}

type consumerTaskArgs struct {
}

func (c *consumerTaskArgs) IsTask() bool { return true }

func (p *CmdWrapper) startBufConsumer(ctx context.Context) {
	consumerFuncExecute := func(w *wp.Worker, wt *wp.WorkerTask) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				data, err := p.buffer.Read()

				if err != nil {
					return err
				}

				_, err = p.bufPipeInPoint.Write(data)
				if err != nil {
					return err
				}
			}
		}
	}

	consumerFuncComplete := func(*wp.Worker, *wp.WorkerTask) {}
	consumerFuncError := func(w *wp.Worker, wt *wp.WorkerTask, err error) { fmt.Println(err) }

	workerpool := wp.MakeWorkerPool(ctx)
	taskArgs := consumerTaskArgs{}
	workerpool.AddWorkers(uint(2))
	task := wp.WorkerTask{Args: &taskArgs,
		Execute:    consumerFuncExecute,
		OnComplete: consumerFuncComplete,
		OnError:    consumerFuncError,
	}
	workerpool.AddTask(&task)
	workerpool.StartWork(ctx)
}

func (p *CmdWrapper) startStdinConsumer(ctx context.Context) {

	if p.inPoint == nil {
		return
	}
	consumerFuncExecute := func(w *wp.Worker, wt *wp.WorkerTask) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				data := make([]byte, 4096)
				n, err := p.inPoint.Read(data)
				data = data[:n]
				if err == io.EOF {
					return io.EOF
				}

				p.buffer.Write(data)
			}
		}
	}

	consumerFuncComplete := func(*wp.Worker, *wp.WorkerTask) {}
	consumerFuncError := func(w *wp.Worker, wt *wp.WorkerTask, err error) { fmt.Println(err) }

	workerpool := wp.MakeWorkerPool(ctx)
	taskArgs := consumerTaskArgs{}
	workerpool.AddWorkers(uint(2))
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
	if p.inPoint != nil {
		fmt.Println("CmdWrapper closing stdin")
		e1 := p.inPoint.Close()
		return e1
	} else {
		return errors.New("inPoint is nil for cmd")
	}
}

func (p *CmdWrapper) CloseStdout() error {
	if p.outPoint != nil {
		fmt.Println("CmdWrapper closing stdout")
		e2 := p.outPoint.Close()
		return e2
	} else {
		return errors.New("outPoint is nil for cmd")
	}
}

func (p *CmdWrapper) CloseBufStdin() error {
	if p.bufPipeInPoint != nil {
		e3 := p.bufPipeInPoint.Close()
		return e3
	} else {
		return errors.New("bufInPoint is nil for cmd")
	}
}
func (p *CmdWrapper) CloseBufStdout() error {
	if p.bufPipeOutPoint != nil {
		e4 := p.bufPipeOutPoint.Close()
		return e4
	} else {
		return errors.New("bufOutPoint is nil for cmd")
	}
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
