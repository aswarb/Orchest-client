package taskParser

import (
	//"fmt"
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

func (tf *TaskFile) GetTaskChain() []*TomlTask {
	orderedTasks := []*TomlTask{}

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
			orderedTasks = append(orderedTasks, tf.GetTaskByUid(k))
		}
	}

	return orderedTasks
}

func GetTomlTaskArray[T any](path string, holderStruct *T) (*T, error) {
	data, _ := os.ReadFile(path)
	fileContents := string(data)

	_, err := toml.Decode(fileContents, holderStruct)
	return holderStruct, err
}
