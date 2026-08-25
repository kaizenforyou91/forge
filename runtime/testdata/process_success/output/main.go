package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

const oversizedOutputBytes = 1*1024*1024 + 4096

func main() {
	fmt.Println("fixture=process-output")
	fmt.Fprintln(os.Stderr, "fixture=process-output-stderr")
	emit(os.Stdout, 'O', oversizedOutputBytes)
	emit(os.Stderr, 'E', oversizedOutputBytes)
}

func emit(writer io.Writer, value byte, total int) {
	chunk := bytes.Repeat([]byte{value}, 4096)
	for total > 0 {
		write := len(chunk)
		if write > total {
			write = total
		}
		if _, err := writer.Write(chunk[:write]); err != nil {
			return
		}
		total -= write
	}
}
