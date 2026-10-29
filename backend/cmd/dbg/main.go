package main

import (
	"fmt"

	"github.com/tafakorpardazan/huginn/internal/normalize"
)

func main() {
	block := "شرکت صدف 2\nتولید اسباب بازی\nsvg031-35314946\nsvginfo\\@sadaf2.ir\nsvg اصفهان، بلوار ارغوانیه"
	cleaned := normalize.CleanSourceText(block)
	fmt.Printf("cleaned:\n%s\n", cleaned)
	fmt.Println("has backslash-at:", indexOf(cleaned, `\@`))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
