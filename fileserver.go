package main

import (
	"net/http"
)

type fileHandler string

func FileServer(dir string) http.Handler {
	return fileHandler(dir)
}

func (self fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	http.FileServer(http.Dir(self)).ServeHTTP(w, r)
}
