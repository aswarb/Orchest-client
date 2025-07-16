package main

import (
	"context"
	"fmt"
	"orchest-client/internal/task"
	"os"
)

func main() {
	wd, _ := os.Getwd()
	ctx, cancelFunc := context.WithCancel(context.Background())
	tomlPath := fmt.Sprintf("%s%s", wd, "/testCases/test1.orchest.task.toml")
	fmt.Println(tomlPath)
	tasks := task.GetTomlTaskArray(tomlPath)
	fmt.Println("TomlTasks:", tasks)
	for _, task := range tasks {
		fmt.Println("Task:", task)
	}
	taskManager := task.GetTaskManagerFromToml(tasks)
	fmt.Println(taskManager)
	taskErr := taskManager.StartTask(ctx)
	fmt.Println(taskErr)
	cancelFunc()
}
