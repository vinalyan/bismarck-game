package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("🚀 Bismarck Game Backend starting...")

    server := NewServer(":8080")

	log.Printf("🌐 Server starting on %s", server.Addr)
	log.Printf("✅ Health check available at http://localhost%s/health", server.Addr)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}
