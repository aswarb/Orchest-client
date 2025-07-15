package task

import (
	"fmt"
	"io"
	"os/exec"
	"time"
)

func CreateTaskExecutor() *TaskExecutor {
	buckets := make(map[string][]byte)
	return &TaskExecutor{buckets: buckets}
}

type TaskExecutor struct {
	buckets map[string][]byte
}

func (t *TaskExecutor) PushToBucket(id string, input []byte) {
	if bucket, exists := t.buckets[id]; exists {
		for _, b := range input {
			bucket = append(bucket, b)
		}
		t.buckets[id] = bucket
	}
}

func (t *TaskExecutor) ExecuteTask(task *Task, nextTasks []*Task) int {
	var cmdInReader, manualOutReader io.ReadCloser
	var manualInWriter, cmdOutWriter io.WriteCloser

	cmd := exec.Command(task.Executable, task.Args...)
	if task.ReadStdin {
		cmdInReader, manualInWriter = io.Pipe()
		cmd.Stdin = cmdInReader
	}
	if task.GiveStdout {
		manualOutReader, cmdOutWriter = io.Pipe()
		cmd.Stdout = cmdOutWriter
	}

	exhaustBucket := func(writer io.WriteCloser, bucket []byte) {
		for len(bucket) > 0 {
			n, err := writer.Write(bucket)
			bucket = bucket[n:]
			if err != nil {
				return
			}
		}
	}
	exhaustStdout := func(reader io.ReadCloser, buckets [][]byte) {
		var err error
		var n int
		for {
			buf := make([]byte, 4096)
			n, err = reader.Read(buf)
			if err != nil {
				break
			}
			buf = buf[:n]
			for i := range buckets {
				buckets[i] = append(buckets[i], buf...)
			}
		}
	}

	if bucket, bucketExists := t.buckets[task.GetUid()]; bucketExists && task.ReadStdin {
		go exhaustBucket(manualInWriter, bucket)
	}

	for _, nUid := range task.GetNext() {
		allBuckets := []([]byte){}
		if bucket, bucketExists := t.buckets[nUid]; bucketExists {
			allBuckets = append(allBuckets, bucket)
		}

		go exhaustStdout(manualOutReader, allBuckets)
	}

	timeout_func := func(time_ms int, cmd *exec.Cmd) {
		time.Sleep(time.Duration(time_ms) * time.Millisecond)
		if !cmd.ProcessState.Exited() {
			cmd.Process.Kill()
		}
	}
	fmt.Println("Starting Delay for Task", task.GetUid(), " of ", task.Delay, " Milliseconds")
	time.Sleep(time.Duration(task.Delay) * time.Millisecond)
	fmt.Println("Starting Task", task.GetUid())
	go timeout_func(int(task.Timeout), cmd)
	cmd.Run()

	return 0
}
