package main

import (
	"fmt"
	"markdown-pdf-go/html"
)

func main() {
	source := "Test.md"
	dest := "Test.html"
	content, err := html.ReadFile(source)
	if err != nil {
		fmt.Println(err)
		return
	}
	mdc := &html.Markdown{}
	htmlContent := mdc.Html(content)
	err = html.WriteFile(dest, htmlContent)
	if err != nil {
		fmt.Println(err)
		return
	}
}
