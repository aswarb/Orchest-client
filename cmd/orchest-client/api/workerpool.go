package api

import (
	"context"
)

type TaskArgs interface {
	getTaskCount() uint
}

type WorkerTask struct {
	args       TaskArgs
	Execute    func() error
	OnComplete func()
	OnError    func(error)
}

type Worker struct {
	taskTimeout    uint64
	taskChannel    chan *WorkerTask
	shouldContinue bool
}

func (w *Worker) Run(ctx context.Context) {
	for w.shouldContinue {
		select {
		case <-ctx.Done():
			return
		case task := <-w.taskChannel:
			err := task.Execute()
			if err == nil {
				task.OnComplete()
			} else {
				task.OnError(err)
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

func (p *WorkerPool) addWorkers(num uint) {
	for i := uint(0); i < num; i++ {
		newWorker := &Worker{
			taskTimeout:    100,
			taskChannel:    p.taskQueue,
			shouldContinue: true,
		}
		p.allWorkers = append(p.allWorkers, newWorker)
	}
}

func (p *WorkerPool) startWork(parentContext context.Context) {

	go func() {
		<-parentContext.Done()
		p.cancelFunc()
	}()

	for i, _ := range p.allWorkers {
		go p.allWorkers[i].Run(p.context)
	}
}
func (p *WorkerPool) stopWork() {
	p.cancelFunc()
}

func (p *WorkerPool) addTask(task *WorkerTask) {
	p.taskQueue <- task
}
