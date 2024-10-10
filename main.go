package main

import (
	"os"

	"github.com/FG420/go-block/cli"
)

func main() {
	defer os.Exit(0)
	// Mian()
	cmd := cli.CommandLine{}
	cmd.Run()

}
