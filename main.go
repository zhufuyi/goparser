package main

import (
	"os"

	"github.com/zhufuyi/goparser/commands"
)

func main() {
	rootCMD := commands.NewRootCMD()
	if err := rootCMD.Execute(); err != nil {
		rootCMD.PrintErrln("Error:", err)
		os.Exit(1)
	}
}
