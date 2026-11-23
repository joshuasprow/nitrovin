package ui

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestStylesheetBraces: every brace of ui.css closes one it opened. A
// closing brace left behind when its block's opening line is deleted
// makes a browser drop the rule after it, in silence, and breaks the
// frames scripts/canvas.sh builds from the stylesheet.
func TestStylesheetBraces(t *testing.T) {
	src, err := os.ReadFile("static/ui.css")
	if err != nil {
		t.Fatal(err)
	}
	// A comment keeps its line breaks, so a line number stays the file's.
	css := regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllStringFunc(string(src), func(c string) string {
		return strings.Repeat("\n", strings.Count(c, "\n"))
	})
	depth := 0
	for n, line := range strings.Split(css, "\n") {
		for _, c := range line {
			switch c {
			case '{':
				depth++
			case '}':
				depth--
			}
			if depth < 0 {
				t.Fatalf("ui.css:%d: a closing brace with no block open", n+1)
			}
		}
	}
	if depth != 0 {
		t.Errorf("ui.css: %d block(s) never closed", depth)
	}
}
