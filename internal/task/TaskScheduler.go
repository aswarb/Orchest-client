package task

import (
	"fmt"
	"maps"
)

type TaskScheduler struct {
	originalCountMap map[string]int
	currentCountMap  map[string]int
	zeroDegreeIds    []string
}

func (t *TaskScheduler) updateZeroDegreeIDs() {
	zeroFilterFunc := func(k string, v int) bool { return v == 0 }
	filteredMap := FilterMap(t.currentCountMap, zeroFilterFunc)

	for k := range filteredMap {
		delete(t.currentCountMap, k)
		t.zeroDegreeIds = append(t.zeroDegreeIds, k)
	}
}

func (t *TaskScheduler) GetNext() (string, error) {
	if len(t.zeroDegreeIds) == 0 {
		t.updateZeroDegreeIDs()
	}
	if len(t.zeroDegreeIds) > 0 {
		ret := t.zeroDegreeIds[0]
		t.zeroDegreeIds = t.zeroDegreeIds[1:]
		return ret, nil
	}
	return "", fmt.Errorf("Task Scheduler cannot get next task. No more nodes with in degree = 0")
}

func (t *TaskScheduler) MarkDone(id string, nextIds []string) {
	for _, id := range nextIds {
		if count, exists := t.currentCountMap[id]; exists && count > 0 {
			count--
			t.currentCountMap[id] = count
		}

		if count, exists := t.currentCountMap[id]; exists && count == 0 {
			t.zeroDegreeIds = append(t.zeroDegreeIds, id)
			delete(t.currentCountMap, id)
		}
	}
}

func (t *TaskScheduler) Reset() {
	t.zeroDegreeIds = []string{}
	newmap := make(map[string]int)
	maps.Copy(newmap, t.originalCountMap)
	t.currentCountMap = newmap
}
