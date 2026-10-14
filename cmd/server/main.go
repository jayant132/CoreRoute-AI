package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/jayant132/CoreRoute-AI/internal/cache"
	"github.com/jayant132/CoreRoute-AI/internal/proxy"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Warning: .env file not found. Relying on system environment variables.")
	}

	ollamaURL := "http://localhost:11434"
	groqAPIKey := os.Getenv("GROQ_API_KEY")
	if groqAPIKey == "" {
		log.Fatal("❌ ERROR: GROQ_API_KEY is not set in the environment or .env file.")
	}

	// 1. Initialize our In-Memory Semantic Cache
	semanticCache := cache.NewSemanticCache() // Wait, let's match the function name from semantic_cache.go (NewSemanticCache)
	// Let's make sure it calls cache.NewSemanticCache()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy", "service": "CoreRoute-AI"}`))
	})

	// 2. Initialize Resilient Proxy with Cache dependency injected
	resilientProxy := proxy.NewResilientProxy(ollamaURL, groqAPIKey, semanticCache)

		http.Handle("/v1/", resilientProxy)
	port := ":8080"
	fmt.Printf("🚀 CoreRoute-AI Gateway starting on port %s...\n", port)
	fmt.Printf("⚡ Semantic Caching & Resilience Engine Active\n")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}