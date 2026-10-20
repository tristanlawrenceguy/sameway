package main

import (
	"fmt"
	"strings"
)

func main() {
	s := `<span class="sw-detail__when sw-muted sw-small"><span class="sw-visually-hidden">created at </span><time datetime="2026-10-20T14:05:32Z" title="Tuesday 20 October 2026 at 4:05pm">today at 4:05pm</time> by an agent</span>`
	fmt.Println("Contains sw-detail__when:", strings.Contains(s, `class="sw-detail__when"`))
	fmt.Println("!Contains check:", !strings.Contains(s, `class="sw-detail__when"`) && !strings.Contains(s, "Created at"))
}
