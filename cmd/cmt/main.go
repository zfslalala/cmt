package main

import (
	"os"
	"path/filepath"
)

func main() {
	if filepath.Base(os.Args[0]) == "gmt" {
		executeGMT()
		return
	}

	executeCMT()
}
