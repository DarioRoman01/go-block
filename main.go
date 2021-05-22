package main

import (
	"os"

	"github.com/Haizza1/go-block/cli"
)

func main() {
	defer os.Exit(0)
	cli := cli.CommandLine{}
	cli.Run()
}
