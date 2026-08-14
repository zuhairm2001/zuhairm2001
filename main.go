package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/zuhairm2001/about-me/internal/handlers"
)

func main() {
	devMode := flag.Bool("dev", false, "enable template and browser hot reloading")
	flag.Parse()

	handlers.SetDevelopmentMode(*devMode)

	mux := http.NewServeMux()
	fs := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	mux.Handle("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *devMode {
			w.Header().Set("Cache-Control", "no-store")
		}
		fs.ServeHTTP(w, r)
	}))

	mux.HandleFunc("/", handlers.IndexHandler)
	mux.HandleFunc("/writings", handlers.WritingsHandler)
	mux.HandleFunc("/writing/", handlers.WritingHandler)

	var server http.Handler = mux
	if *devMode {
		reloader := newLiveReloader()
		mux.Handle("/__dev/reload", reloader)
		go reloader.watch([]string{"static", "styles", "templates"})
		server = injectLiveReload(mux)
		log.Println("Development mode enabled; watching static, styles, and templates")
	}

	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", server))
}
