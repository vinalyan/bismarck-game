package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	fmt.Println("🚀 Bismarck Game Backend starting...")

	router := mux.NewRouter()

	// Basic health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok", "service": "bismarck-game"}`))
	}).Methods("GET")

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
	}).Methods("GET")

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Printf("🌐 Server starting on %s", server.Addr)
	log.Printf("✅ Health check available at http://localhost%s/health", server.Addr)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}
