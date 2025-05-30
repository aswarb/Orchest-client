package task

import ()

type TaskEngine struct {
	resolver *DAGResolver
}

// Creates pipes for adjacent nodes
func (t *TaskEngine) createPipes(tasks []Task) {
}
