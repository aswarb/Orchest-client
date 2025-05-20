package taskParser

import (
	"io"
	"os/exec"
)

type Task struct {
	uid        string
	name       string
	command    *exec.Cmd
	timeout    uint64
	delay      uint64
	next       []*Task
	givestdout bool
	readstdin  bool
}

func (t *Task) Next() []*Task {
	return t.next
}

func (t *Task) GetCmd() *exec.Cmd {
	return t.command
}

func (t *Task) preExecuteSetup() {
	destWriters := []io.Writer{}

	for _, nextTask := range t.next {
		if nextTask.WantsStdin() && t.givestdout {
			pipeRead, pipeWrite := io.Pipe()
			destWriters = append(destWriters, pipeWrite)
			nextTask.command.Stdin = pipeRead
		}
	}

	if t.givestdout && len(destWriters) > 0 {
		multiWriter := io.MultiWriter(destWriters...)
		t.command.Stdout = multiWriter
	}
}

func (t *Task) ExecuteBlocking() {
	t.preExecuteSetup()
	t.command.Start()
}

func (t *Task) Execute() {
	t.preExecuteSetup()
	t.command.Run()
}

func (t *Task) AddNextTasks(tasks ...*Task) {
	for _, task := range tasks {
		t.next = append(t.next, task)
	}
}

func (t *Task) SetNextTasks(tasks []*Task) {
	t.next = tasks
}

func (t *Task) WantsStdin() bool {
	return t.readstdin
}

func (t *Task) GivesStdout() bool {
	return t.givestdout
}

func GetTask(executable string, args []string, uid string, timeout uint64,
	delay uint64, next []*Task, givestdout bool, readstdin bool) *Task {

	cmd := exec.Command(executable, args...)

	task := &Task{
		command:    cmd,
		uid:        uid,
		timeout:    timeout,
		delay:      delay,
		next:       next,
		givestdout: givestdout,
		readstdin:  readstdin,
	}

	return task
}

// Topological sort of a Directed Acyclic Graph
func GetSerialExecuteOrder(startingNode *Task) {

}
