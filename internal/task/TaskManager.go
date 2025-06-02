package task

import (
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

func (t *TaskManager) startTask() {
	//
}

