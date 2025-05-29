package task

import (
	"io"
	"os/exec"
)

type Task interface {
	ExecuteBlocking()
	Execute()
	GetUid() string
	GetTimeout() uint64
	GetDelay() uint64
	GetGiveStdout() bool
	GetReadStdin() bool
	GetStdin() io.Reader
	SetStdin(io.Reader)
	GetStdout() io.Writer
	SetStdout(io.Writer)
}

type ParallelTask struct {
	uid        string
	name       string
	taskChain  []Task
	timeout    uint64
	delay      uint64
	givestdout bool
	readstdin  bool
}

type SingleTask struct {
	uid        string
	name       string
	command    *exec.Cmd
	timeout    uint64
	delay      uint64
	giveStdout bool
	readStdin  bool
}

func (t *SingleTask) SetStdout(writer io.Writer) { t.command.Stdout = writer }
func (t *SingleTask) SetStdin(reader io.Reader)  { t.command.Stdin = reader }

func (t *SingleTask) GetCmd() *exec.Cmd   { return t.command }
func (t *SingleTask) GetUid() string      { return t.uid }
func (t *SingleTask) GetName() string     { return t.name }
func (t *SingleTask) GetTimeout() uint64  { return t.timeout }
func (t *SingleTask) GetDelay() uint64    { return t.delay }
func (t *SingleTask) GetGiveStdout() bool { return t.giveStdout }
func (t *SingleTask) GetReadStdin() bool  { return t.readStdin }

func (t *SingleTask) ExecuteBlocking() {
	t.command.Start()
}

func (t *SingleTask) Execute() {
	t.command.Run()
}

func (t *SingleTask) WantsStdin() bool {
	return t.readStdin
}

func (t *SingleTask) GivesStdout() bool {
	return t.giveStdout
}
func (t *SingleTask) GetStdin() io.Reader {
	return t.command.Stdin
}

func (t *SingleTask) GetStdout() io.Writer {
	return t.command.Stdout
}

func (t *SingleTask) FillFromToml() {

}

func GetSingleTask(name string, executable string, args []string, uid string, timeout uint64,
	delay uint64, givestdout bool, readstdin bool) *SingleTask {

	cmd := exec.Command(executable, args...)

	task := SingleTask{
		name:       name,
		command:    cmd,
		uid:        uid,
		timeout:    timeout,
		delay:      delay,
		giveStdout: givestdout,
		readStdin:  readstdin,
	}

	return &task
}

// Toplogical sort of a Directed Acyclic Graph
func GetSerialExecuteOrder(startingNode *Task) {

}
