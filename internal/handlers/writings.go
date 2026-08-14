package handlers

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

type PostMeta struct {
	Title string `yaml:"title"`
}

type PostSummary struct {
	Title string
	Date  string
	Slug  string
}

type WritingsPageData struct {
	Posts []PostSummary
}

type WritingPageData struct {
	Title   string
	Date    string
	Content template.HTML
}

var writingsTmpl = template.Must(template.ParseFiles("templates/writings.html"))
var writingTmpl = template.Must(template.ParseFiles("templates/writing.html"))

func WritingsHandler(w http.ResponseWriter, r *http.Request) {
	var posts []PostSummary

	// Read all date directories in writings/
	entries, err := os.ReadDir("writings")
	if err != nil {
		http.Error(w, "Failed to read writings directory", http.StatusInternalServerError)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		date := entry.Name()
		dateDir := filepath.Join("writings", date)

		// Find markdown files in the date directory
		mdFiles, err := filepath.Glob(filepath.Join(dateDir, "*.md"))
		if err != nil || len(mdFiles) == 0 {
			continue
		}

		for _, mdFile := range mdFiles {
			// Read and parse frontmatter
			file, err := os.Open(mdFile)
			if err != nil {
				continue
			}

			var meta PostMeta
			_, err = frontmatter.Parse(file, &meta)
			file.Close()
			if err != nil {
				continue
			}

			// Get slug from filename (without .md extension)
			slug := strings.TrimSuffix(filepath.Base(mdFile), ".md")

			posts = append(posts, PostSummary{
				Title: meta.Title,
				Date:  date,
				Slug:  slug,
			})
		}
	}

	tmpl, err := templateFor("templates/writings.html", writingsTmpl)
	if err != nil {
		http.Error(w, "Failed to parse writings template", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, WritingsPageData{Posts: posts}); err != nil {
		http.Error(w, "Failed to render writings", http.StatusInternalServerError)
	}
}

func WritingHandler(w http.ResponseWriter, r *http.Request) {
	// Parse URL: /writing/{date}/{slug}
	path := strings.TrimPrefix(r.URL.Path, "/writing/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) != 2 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	date := parts[0]
	slug := parts[1]

	// Construct file path
	mdPath := filepath.Join("writings", date, slug+".md")

	// Read the markdown file
	file, err := os.Open(mdPath)
	if err != nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Parse frontmatter
	var meta PostMeta
	content, err := frontmatter.Parse(file, &meta)
	if err != nil {
		http.Error(w, "Failed to parse post", http.StatusInternalServerError)
		return
	}

	// Convert markdown to HTML
	htmlContent := mdToHTML(content)

	tmpl, err := templateFor("templates/writing.html", writingTmpl)
	if err != nil {
		http.Error(w, "Failed to parse writing template", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, WritingPageData{
		Title:   meta.Title,
		Date:    date,
		Content: template.HTML(htmlContent),
	}); err != nil {
		http.Error(w, "Failed to render writing", http.StatusInternalServerError)
	}
}

func mdToHTML(md []byte) []byte {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}
