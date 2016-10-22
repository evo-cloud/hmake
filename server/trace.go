package server

import "net/http"

func (s *Server) traceRequest(w http.ResponseWriter, r *http.Request) http.ResponseWriter {
	return w
}
