package workerpool

import (
	"context"
	"fmt"
)

type TaskArgs interface {
	IsTask() bool
}

type WorkerTask struct {
	Args       TaskArgs
	Execute    func(*Worker, *WorkerTask) error
	OnComplete func(*Worker, *WorkerTask)
	OnError    func(*Worker, *WorkerTask, error)
}

func (wt *WorkerTask) GetArgs() TaskArgs {
	return wt.Args
}

type Worker struct {
	taskTimeout    uint64
	taskChannel    chan *WorkerTask
	shouldContinue bool
}

func (w *Worker) GetTimeout() uint64 {
	return w.taskTimeout
}

func (w *Worker) GetTaskChan() chan *WorkerTask {
	return w.taskChannel
}

func (w *Worker) Run(ctx context.Context) {
	for w.shouldContinue {
		select {
		case <-ctx.Done():
			return
		case task := <-w.taskChannel:
			err := task.Execute(w, task)
			if err == nil {
				task.OnComplete(w, task)
			} else {
				task.OnError(w, task, err)
			}
		}

	}
}

func (w *Worker) Stop() {
	w.shouldContinue = false
}

func MakeWorkerPool(ctx context.Context) WorkerPool {

	thisContext, cancelFunc := context.WithCancel(ctx)
	taskQueue := make(chan *WorkerTask)
	workerSlice := ([]*Worker{})[:]

	wp := WorkerPool{
		allWorkers: workerSlice,
		taskQueue:  taskQueue,
		context:    thisContext,
		cancelFunc: cancelFunc,
	}
	return wp
}

type WorkerPool struct {
	allWorkers []*Worker
	taskQueue  chan *WorkerTask
	context    context.Context
	cancelFunc func()
}

func (p *WorkerPool) AddWorkers(num uint) {
	for i := uint(0); i < num; i++ {
		newWorker := &Worker{
			taskTimeout:    100,
			taskChannel:    p.taskQueue,
			shouldContinue: true,
		}
		p.allWorkers = append(p.allWorkers, newWorker)
	}
}

func (p *WorkerPool) GetWorkerCount() int { return len(p.allWorkers) }

func (p *WorkerPool) RemoveWorkers(n int) error {
	if len(p.allWorkers) > n {
		return fmt.Errorf("Cannot remove %d workers, pool only contains %d workers total", n, len(p.allWorkers))
	}

	toRemove := p.allWorkers[:n]
	toKeep := p.allWorkers[n:]

	p.allWorkers = toKeep

	for _, workerPointer := range toRemove {
		workerPointer.Stop()
	}

	return nil
}

func (p *WorkerPool) StartWork(parentContext context.Context) {

	go func() {
		<-parentContext.Done()
		p.cancelFunc()
	}()

	for i, _ := range p.allWorkers {
		go p.allWorkers[i].Run(p.context)
	}
}
func (p *WorkerPool) StopWork() {
	p.cancelFunc()
}

func (p *WorkerPool) AddTask(task *WorkerTask) {
	p.taskQueue <- task
}

func (p *WorkerPool) GetTaskChan() chan *WorkerTask {
	return p.taskQueue
}
