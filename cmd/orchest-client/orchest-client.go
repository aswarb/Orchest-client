package main

import (
	"strconv"
)

func main() {
	var arr []Command
	for i := 0; i < 100; i++ {

		t := START_PROCESS_MESSAGE
		if i%2 == 1 {
			t = STOP_PROCESS_MESSAGE
		}

		cmd := Command{messageType: t, payload: strconv.Itoa(i)}
		arr = append(arr, cmd)
	}

	for _, command := range arr {
		command.Execute()
	}

}
