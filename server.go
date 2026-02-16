package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed frontend
var frontendFS embed.FS

// Server is the main HTTP server.
type Server struct {
	mux *http.ServeMux
}

// NewServer creates a configured HTTP server.
func NewServer() *Server {
	s := &Server{
		mux: http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// Frontend static files
	frontendDir, _ := fs.Sub(frontendFS, "frontend")
	fileServer := http.FileServer(http.FS(frontendDir))
	s.mux.Handle("GET /", fileServer)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
