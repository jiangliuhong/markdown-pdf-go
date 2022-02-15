package main

import (
	"markdown-pdf-go/web/handle"
	"net/http"
)

func main() {
	http.HandleFunc("/", handle.Index)
	http.ListenAndServe(":8888", nil)
}
