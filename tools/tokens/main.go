// Command tokens regenerates design/tokens/tokens.css from tokens.json.
package main

import (
	"fmt"
	"os"

	"github.com/sameway-dev/sameway/internal/tokens"
)

func main() {
	src, err := os.ReadFile("design/tokens/tokens.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "tokens: run from the repository root:", err)
		os.Exit(1)
	}
	css, err := tokens.Generate(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tokens:", err)
		os.Exit(1)
	}
	if err := os.WriteFile("design/tokens/tokens.css", []byte(css), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "tokens:", err)
		os.Exit(1)
	}
	fmt.Println("wrote design/tokens/tokens.css")
}
