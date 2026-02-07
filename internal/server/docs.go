package server

import (
	"embed"
	"io/fs"
	"net/http"
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
	return http.StripPrefix("/docs/", http.FileServer(http.FS(root)))
}
