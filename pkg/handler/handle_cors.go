package handler

import "net/http"

type WithCORSHandler struct {
	next http.Handler
}

func NewWithCORSHandler(handler http.Handler) *WithCORSHandler {
	return &WithCORSHandler{next: handler}
}

func (h *WithCORSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	h.next.ServeHTTP(w, r)
}
