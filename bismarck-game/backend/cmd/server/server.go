package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// NewServer constructs and returns an *http.Server with all routes configured.
func NewServer(addr string) *http.Server {
	router := mux.NewRouter()

	// Basic health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok", "service": "bismarck-game"}`))
	}).Methods(http.MethodGet)

	// API info
	router.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"name": "Bismarck Game API",
			"version": "0.1.0",
			"endpoints": {
				"health": "/health",
				"api_info": "/api"
			}
		}`))
	}).Methods(http.MethodGet)

	return &http.Server{
		Addr:    addr,
		Handler: router,
	}
}
