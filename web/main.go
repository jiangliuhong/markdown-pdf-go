package main

import (
	"markdown-pdf-go/web/handle"
	"net/http"
)

func main() {
	http.HandleFunc("/", handle.Index)
	http.HandleFunc("/demo", handle.Demo)
	http.ListenAndServe(":8888", nil)
}
