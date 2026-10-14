package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/jayant132/CoreRoute-AI/internal/proxy"
)

func main() {
	ollamaURL := "http://localhost:11434"

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy", "service": "CoreRoute-AI"}`))
	})

	ollamaProxy := proxy.NewOllamaProxy(ollamaURL)

	http.Handle("/v1/", http.StripPrefix("/v1", ollamaProxy))

	port := ":8080"
	fmt.Printf("🚀 CoreRoute-AI Gateway starting on port %s...\n", port)
	fmt.Printf("🔄 Forwarding AI traffic downstream to Ollama at %s\n", ollamaURL)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}