package task

import (
	"context"
	"fmt"
)

func FilterMap[K comparable, V any](m map[K]V, isValid func(K, V) bool) map[K]V {
	result := make(map[K]V)
	for k, v := range m {
		if isValid(k, v) {
			result[k] = v
		}
	}
	return result
}

func MapHasKey[K comparable, V any](m map[K]V, key K) bool {
	_, ok := m[key]
	return ok

}

type TaskManager struct {
	engine   *TaskEngine
	resolver *DAGResolver
}

func (t *TaskManager) StartTask(ctx context.Context) {
	t.engine.ExecuteTasksInOrder(ctx)
}

func GetTaskManagerFromToml(sourceTasks []TomlTask, sourceSegments []TomlSegment) *TaskManager {
	tasks := []Node{}
	for _, target := range sourceTasks {
		switch target.(type) {
		case *SingleTomlTask:
			t := target.(*SingleTomlTask)
			task := GetTask(t.Uid,
				t.Name,
				t.Command,
				t.Args,
				uint64(t.Timeout),
				uint64(t.Delay),
				t.Next,
				t.GiveStdout,
				t.ReadStdin)
			tasks = append(tasks, task)
		default:
			continue
		}
	}

	resolver := MakeDAGResolver(tasks)
	scheduler := CreateTaskScheduler(resolver.CountIncomingEdges(nil))
	executor := CreateTaskExecutor()
	manager := TaskManager{
		resolver:  resolver,
		scheduler: scheduler,
		executor:  executor,
	}

	return &manager
}
