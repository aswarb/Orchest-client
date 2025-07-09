package task

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"orchest-client/templates/task"
	"os"
	"regexp"
)

type TomlTask interface {
	GetUid() string
	GetNext() []string
	ToString() string
}

type SingleTomlTask struct {
	Uid        string   `toml:"uid"`
	Name       string   `toml:"name"`
	Command    string   `toml:"command"`
	Args       []string `toml:"args"`
	Timeout    int      `toml:"timeout"`
	Delay      int      `toml:"delay"`
	Next       []string `toml:"next"`
	GiveStdout bool     `toml:"givestdout"`
	ReadStdin  bool     `toml:"readstdin"`
}

func (t *SingleTomlTask) GetUid() string    { return t.Uid }
func (t *SingleTomlTask) GetNext() []string { return t.Next }
func (t *SingleTomlTask) ToString() string {
	template, _ := taskTemplate.GetTemplate(taskTemplate.SINGLE_TASK)
	return string(template)

}

func GetTomlTaskArray(path string) []TomlTask {
	data, _ := os.ReadFile(path)
	fileContents := string(data)

	TableHeaderPattern, _ := regexp.Compile(`\[\[(Task)\]\]`)
	indices := TableHeaderPattern.FindAllStringIndex(fileContents, -1)
	blobs := make(map[string][]string)
	for num, pair := range indices {
		var blob string
		if num == len(indices)-1 {
			blob = fileContents[pair[1]:]
		} else {
			blob = fileContents[pair[1]:indices[num+1][0]]
		}
		header := fileContents[pair[0]:pair[1]]
		_, ok := blobs[header]
		if !ok {
			blobs[header] = []string{}
		}
		blobs[header] = append(blobs[header], blob)
	}

	tasks := []TomlTask{}
	for k, _ := range blobs {
		for _, blob := range blobs[k] {
			switch k {
			case "[[Task]]":
				var holder SingleTomlTask
				_, _ = toml.Decode(blob, &holder)
				tasks = append(tasks, &holder)

			default:
				continue
			}
		}
	}
	for _, t := range tasks {
		fmt.Println(t)
	}
	return tasks
}
