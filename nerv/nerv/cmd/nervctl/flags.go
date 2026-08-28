package main

import (
	"flag"
	"log"
)

func newFlagSet(name string) *flag.FlagSet {
	return flag.NewFlagSet(name, flag.ExitOnError)
}

func mustParse(fs *flag.FlagSet, args []string) {
	if err := fs.Parse(args); err != nil {
		log.Fatalf("nervctl %s: parsing flags: %v", fs.Name(), err)
	}
}
