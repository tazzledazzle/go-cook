package main

import (
	"fmt"
	"os"

	"github.com/tazzledazzle/go-cook/nerv-ecosystem/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
