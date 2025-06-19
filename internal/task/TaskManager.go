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
	segments := []Segment{}
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

	for _, target := range sourceSegments {
		switch target.(type) {
		case *ParallelTomlSegment:
			fmt.Println(target)
			s := target.(*ParallelTomlSegment)
			fmt.Println("createTaskManager:", s.EndUids)
			segment := GetParallelSegment(s.Uid,
				s.Name,
				s.StartUids,
				s.EndUids)
			segments = append(segments, segment)
		default:
			continue
		}
	}

	resolver := MakeDAGResolver(tasks, segments)
	engine := GetTaskEngine(resolver)
	manager := TaskManager{engine: engine, resolver: resolver}

	return &manager
}
