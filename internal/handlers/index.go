package handlers

import (
	"html/template"
	"net/http"
)

var (
	developmentMode bool
	indexTmpl       = template.Must(template.ParseFiles("templates/index.html"))
)

// SetDevelopmentMode enables parsing templates on every request so edits are
// visible without restarting the server.
func SetDevelopmentMode(enabled bool) {
	developmentMode = enabled
}

func templateFor(path string, cached *template.Template) (*template.Template, error) {
	if developmentMode {
		return template.ParseFiles(path)
	}
	return cached, nil
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := templateFor("templates/index.html", indexTmpl)
	if err != nil {
		http.Error(w, "Failed to parse index template", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Failed to render index", http.StatusInternalServerError)
	}
}
