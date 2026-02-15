package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/wiredoor/wiredoor-cli/cmd"
)

func main() {
	out := flag.String("out", "completions", "output dir")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}

	root := cmd.RootCmd()

	// bash
	{
		f, err := os.Create(filepath.Join(*out, "wiredoor.bash"))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := root.GenBashCompletionV2(f, true); err != nil {
			log.Fatal(err)
		}
	}

	// zsh
	{
		f, err := os.Create(filepath.Join(*out, "_wiredoor"))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := root.GenZshCompletion(f); err != nil {
			log.Fatal(err)
		}
	}

	// fish
	{
		f, err := os.Create(filepath.Join(*out, "wiredoor.fish"))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := root.GenFishCompletion(f, true); err != nil {
			log.Fatal(err)
		}
	}
}
