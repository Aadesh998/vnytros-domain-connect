package mcp

import (
	"bytes"
	"embed"
	"html/template"
	"log"
	"net/http"
)

//go:embed html/index.html
var landingFS embed.FS

var landingTmpl = template.Must(template.ParseFS(landingFS, "html/index.html"))

type LandingInfo struct {
	ServerName    string
	ServerVersion string
	WebsiteURL    string
	Resource      string
	AuthServer    string
	MetadataURL   string
	Tools         []ToolDoc
}

func LandingHandler(info LandingInfo) http.Handler {
	info.Tools = append(ToolDocs(), NetToolDocs()...)

	var page bytes.Buffer
	if err := landingTmpl.Execute(&page, info); err != nil {
		log.Printf("ERROR: rendering MCP landing page: %v", err)
	}
	body := page.Bytes()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		if _, err := w.Write(body); err != nil {
			log.Printf("ERROR: writing MCP landing page: %v", err)
		}
	})
}
