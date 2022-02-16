package main

import (
	"fmt"
	"io/ioutil"
	"markdown-pdf-go/html"
)

func main() {
	source := "Test.md"
	dest := "Test.pdf"
	content, err := html.ReadFile(source)
	if err != nil {
		fmt.Println(err)
		return
	}
	mdc := &html.Markdown{}
	pdf := mdc.Pdf(content)
	ioutil.WriteFile(dest, pdf, 0644)
}
