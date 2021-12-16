package main

import (
	"os"

	"github.com/sftpgo/sftpgo-plugin-metadata/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
