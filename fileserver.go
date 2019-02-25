package main

import (
	"net/http"
)

type fileHandler struct {
	root string
}

func FileServer(dir string) http.Handler {
	return &fileHandler{dir}
}

func (f *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	root := f.root
	http.FileServer(http.Dir(root)).ServeHTTP(w, r)
}
