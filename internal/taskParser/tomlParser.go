package taskparser

import (
	"fmt"
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

type TaskFile struct {
	Task []TomlTask `toml:"Task"`
}

func GetTomlTaskArray[T any](holderStruct T) (T, error) {
	wd, _ := os.Getwd()
	data, _ := os.ReadFile(fmt.Sprintf("%s/internal/taskParser/TEMPLATE.orchest.task.toml", wd))
	fileContents := string(data)

	_, err := toml.Decode(fileContents, &holderStruct)

	return holderStruct, err
}
