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
	htmlBody := html.Markdown(content)
	htmlContent := `<html>
<head>
<meta charset="utf-8">
	<link href="https://cdn.bootcdn.net/ajax/libs/github-markdown-css/5.1.0/github-markdown-light.css" media="all" rel="stylesheet" type="text/css" />
</head>
<body>
<article class="markdown-body entry-content" style="padding: 30px;">`
	htmlContent += string(htmlBody)
	htmlContent += `</article>
</body>
</html>`
	//htmlContent := htmlStart + string(htmlBody) + htmlEnd
	err = html.WriteFile(dest, []byte(htmlContent))
	if err != nil {
		fmt.Println(err)
		return
	}
}
