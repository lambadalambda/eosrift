package server

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed docs_static
var docsEmbeddedFS embed.FS

var docsHandler = newDocsHandler()

func serveDocs(w http.ResponseWriter, r *http.Request) {
	docsHandler.ServeHTTP(w, r)
}

func newDocsHandler() http.Handler {
	root, err := fs.Sub(docsEmbeddedFS, "docs_static")
	if err != nil {
		panic(err)
	}
	fileServer := http.StripPrefix("/docs/", http.FileServer(http.FS(root)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, "/docs/")
		if rel == "" {
			fileServer.ServeHTTP(w, r)
			return
		}
		for _, candidate := range docsPathCandidates(rel) {
			if !fs.ValidPath(candidate) {
				continue
			}
			info, err := fs.Stat(root, candidate)
			if err != nil || info.IsDir() {
				continue
			}

			req := r.Clone(r.Context())
			urlCopy := *r.URL
			urlCopy.Path = "/docs/" + candidate
			req.URL = &urlCopy
			fileServer.ServeHTTP(w, req)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func docsPathCandidates(rel string) []string {
	clean := path.Clean("/" + rel)
	normalized := strings.TrimPrefix(clean, "/")

	if normalized == "." || normalized == "" {
		return nil
	}

	candidates := []string{normalized}
	if path.Ext(normalized) == "" {
		candidates = append(candidates, normalized+".html")
	}
	return candidates
}
