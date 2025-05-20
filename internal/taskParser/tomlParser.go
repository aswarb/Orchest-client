package taskParser

import (
	//"fmt"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
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

// NOTE: STILL NEEDS CYCLE DETECTION. HINT: LOOK FOR NO VALID ROOT NODES
func (tf *TaskFile) GetTaskChain() []*TomlTask {
	orderedTasks := []*TomlTask{}
	rootNodes := []*TomlTask{}

	incomingCountMap := make(map[string]int)

	for _, task := range tf.Tasks {

		_, thisTaskInMap := incomingCountMap[task.Uid]
		if !thisTaskInMap {
			incomingCountMap[task.Uid] = 0
		}

		for _, uid := range task.Next {
			if uid == "" {
				continue
			}
			_, ok := incomingCountMap[uid]

			if ok {
				incomingCountMap[uid]++
			} else {
				incomingCountMap[uid] = 1
			}
		}
	}

	for k, v := range incomingCountMap {
		if v == 0 {
			rootNodes = append(rootNodes, tf.GetTaskByUid(k))
		}
	}
	// Kahn's Algorithm: https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm

	inEdgesCounts := make(map[string]int)
	for k, v := range incomingCountMap {
		inEdgesCounts[k] = v
	}
	/*
		for rootIdx, root := range rootNodes {
			orderedTasks := append(orderedTasks, root)
			for _, next := range root.Next {
				//
			}
		}
	*/
	return orderedTasks
}

func GetTomlTaskArray[T any](path string, holderStruct *T) (*T, error) {
	data, _ := os.ReadFile(path)
	fileContents := string(data)

	_, err := toml.Decode(fileContents, holderStruct)
	return holderStruct, err
}

func TomlTasksToTasks(arr []TomlTask) []*Task {

	// Task being waited for : Waiting task
	toBeWired := make(map[string][]*Task)

	tasks := []*Task{}

	for _, el := range arr {
		nextTasks := []*Task{}
		task := GetTask(
			el.GetName(),
			el.GetCommand(),
			el.GetArgs(),
			el.GetUid(),
			uint64(el.GetTimeout()),
			uint64(el.GetDelay()),
			nextTasks,
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
				val = []*Task{task}
				toBeWired[nextUid] = val
			}
		}
	}

	for _, el := range tasks {
		waitingTasks, ok := toBeWired[el.GetUid()]

		if !ok {
			continue
		} else {
			for _, t := range waitingTasks {
				t.AddNextTasks(el)
			}
		}
	}
	fmt.Println((toBeWired))

	return tasks
}
