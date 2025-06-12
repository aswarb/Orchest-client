package taskTemplate

import (
	"embed"
	"fmt"
)

//go:embed *.orchest.task.toml
var TaskTemplate embed.FS

type TaskType string

const (
	SINGLE_TASK      TaskType = "Single"
	PARALLEL_SEGMENT TaskType = "Parallel"
)

func GetTemplate(t TaskType) ([]byte, error) {
	file := ""
	switch t {
	case SINGLE_TASK:
		file = "SINGLE_TASK_TEMPLATE.orchest.task.toml"
	case PARALLEL_SEGMENT:
		file = "PARALLEL_TASK_TEMPLATE.orchest.task.toml"
	default:
		file = ""
	}

	var value []byte
	var err error
	if file != "" {
		value, err = TaskTemplate.ReadFile(file)
	} else {
		value, err = []byte{}, fmt.Errorf("Template for Task of type %s could not be found", string(t))
	}
	return value, err
}
