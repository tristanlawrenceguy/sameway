// Command sameway runs a Sameway workspace: an accessible content system and
// design system that people and AI agents use the same way.
package main

import (
	"os"

	"github.com/tristanlawrenceguy/sameway/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Env{Stdout: os.Stdout, Stderr: os.Stderr}))
}
