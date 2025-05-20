package taskParser

import (
	"io"
	"os/exec"
)

type Task struct {
	command    *exec.Cmd
	uid        string
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

func (t *Task) ExecuteBlocking() {

	t.command.Start()
}

func (t *Task) Execute() {

	t.command.Run()
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

	destWriters := []io.Writer{}

	for _, nextTask := range next {
		if nextTask.WantsStdin() && givestdout {
			pipeRead, pipeWrite := io.Pipe()
			destWriters = append(destWriters, pipeWrite)
			nextTask.command.Stdin = pipeRead
		}
	}

	if givestdout && len(destWriters) > 0 {
		multiWriter := io.MultiWriter(destWriters...)
		cmd.Stdout = multiWriter
	}
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
