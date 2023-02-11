package handle

import (
	"html/template"
	"path/filepath"
)

func init() {

}

// 获取模版
func getTemplate(templateFileName string) *template.Template {
	basePath, _ := filepath.Abs("./web/view")
	templateFullPath := basePath + "/" + templateFileName
	return template.Must(template.ParseFiles(templateFullPath))
}
