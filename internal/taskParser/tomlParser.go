package taskParser

import (
	"github.com/BurntSushi/toml"
	"os"
)

type TomlTask struct {
	Uid        string   `toml:"uid"`
	Name       string   `toml:"name"`
	Command    string   `toml:"command"`
	Args       []string `toml:"args"`
	Timeout    int      `toml:"timeout"`
	Delay      int      `toml:"delay"`
	Next       []string `toml:"next"`
	Givestdout bool     `toml:"givestdout"`
	Readstdin  bool     `toml:"readstdin"`
}

func (t *TomlTask) GetUid() string      { return t.Uid }
func (t *TomlTask) GetName() string     { return t.Name }
func (t *TomlTask) GetCommand() string  { return t.Command }
func (t *TomlTask) GetArgs() []string   { return t.Args }
func (t *TomlTask) GetTimeout() int     { return t.Timeout }
func (t *TomlTask) GetDelay() int       { return t.Delay }
func (t *TomlTask) GetNext() []string   { return t.Next }
func (t *TomlTask) GetGivestdout() bool { return t.Givestdout }
func (t *TomlTask) GetReadstdin() bool  { return t.Readstdin }

type TaskFile struct {
	Tasks []TomlTask `toml:"Task"`
}

func (tf *TaskFile) GetTaskByUid(uid string) *TomlTask {
	for _, task := range tf.Tasks {
		if task.Uid == uid {
			return &task
		}
	}
	return nil
}

func GetTomlTaskArray[T any](path string, holderStruct *T) (*T, error) {
	data, _ := os.ReadFile(path)
	fileContents := string(data)

	_, err := toml.Decode(fileContents, holderStruct)
	return holderStruct, err
}

func TomlTasksToTasks(arr []TomlTask) []Task {

	// Task being waited for : Waiting task
	toBeWired := make(map[string][]Task)

	tasks := []Task{}

	for _, el := range arr {
		//nextTasks := []Task{}
		task := GetSingleTask(
			el.GetName(),
			el.GetCommand(),
			el.GetArgs(),
			el.GetUid(),
			uint64(el.GetTimeout()),
			uint64(el.GetDelay()),
			el.GetGivestdout(),
			el.GetReadstdin(),
		)
		tasks = append(tasks, task)

		for _, nextUid := range el.GetNext() {
			if nextUid == "" {
				continue
			}
			val, ok := toBeWired[nextUid]

			if ok {
				val = append(val, task)
				toBeWired[nextUid] = val
			} else {
				val = []Task{task}
				toBeWired[nextUid] = val
			}
		}
	}

	return tasks
}
