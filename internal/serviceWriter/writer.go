package serviceWriter

import (
	"bufio"
	"fmt"
	"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}
func writeFullFile(data []byte, filepath string) {

	err := os.WriteFile(filepath, data, 0644)
	check(err)

	f, err := os.Create("/tmp/dat2")
	check(err)

	defer f.Close()
}
