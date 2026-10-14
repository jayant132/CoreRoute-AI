package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/jayant132/CoreRoute-AI/internal/proxy"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Warning: .env file not found. Relying on system environment variables.")
	}

	ollamaURL := "http://localhost:9999"
	
	groqAPIKey := os.Getenv("GROQ_API_KEY")
	if groqAPIKey == "" {
		log.Fatal("❌ ERROR: GROQ_API_KEY is not set in the environment or .env file.")
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy", "service": "CoreRoute-AI"}`))
	})

	resilientProxy := proxy.NewResilientProxy(ollamaURL, groqAPIKey)

	http.Handle("/v1/", http.StripPrefix("/v1", resilientProxy))

	port := ":8080"
	fmt.Printf("🚀 CoreRoute-AI Gateway starting on port %s...\n", port)
	fmt.Printf("🛡️ Resilience Engine Active: Primary -> Ollama (%s) | Secondary -> Groq Cloud\n", ollamaURL)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}