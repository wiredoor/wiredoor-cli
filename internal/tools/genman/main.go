package main

import (
	"flag"
	"log"
	"os"

	"github.com/spf13/cobra/doc"

	"github.com/wiredoor/wiredoor-cli/cmd"
)

func main() {
	out := flag.String("out", "man", "output dir")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}

	root := cmd.RootCmd()

	header := &doc.GenManHeader{
		Title:   "WIREDOOR",
		Section: "1",
		Source:  "Wiredoor CLI",
		Manual:  "Wiredoor Manual",
	}

	if err := doc.GenManTree(root, header, *out); err != nil {
		log.Fatal(err)
	}
}
