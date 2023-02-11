package handle

import "net/http"

func Index(w http.ResponseWriter, r *http.Request) {
	template := getTemplate("index.html")
	_ = template.Execute(w, nil)
}
