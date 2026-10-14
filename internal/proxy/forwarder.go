package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/jayant132/CoreRoute-AI/internal/cache"
)

// ResilientProxy manages caching, local Ollama execution, and Groq fallback
type ResilientProxy struct {
	ollamaURL  string
	groqURL    string
	groqAPIKey string
	httpClient *http.Client
	cache      *cache.SemanticCache
}

func NewResilientProxy(ollamaURL, groqAPIKey string, c *cache.SemanticCache) *ResilientProxy {
	return &ResilientProxy{
		ollamaURL:  ollamaURL,
		groqURL:    "https://api.groq.com/openai/v1",
		groqAPIKey: groqAPIKey,
		httpClient: &http.Client{},
		cache:      c,
	}
}

// OpenAIRequest defines the standard JSON structure for chat requests
type OpenAIRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

// ServeHTTP implements the http.Handler interface with Semantic Caching
func (p *ResilientProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Extract the user prompt from the JSON body to check our cache
	var reqPayload OpenAIRequest
	promptText := ""
	if err := json.Unmarshal(bodyBytes, &reqPayload); err == nil && len(reqPayload.Messages) > 0 {
		promptText = reqPayload.Messages[len(reqPayload.Messages)-1].Content
	}

	// 1. CHECK SEMANTIC CACHE (If prompt is valid)
	if promptText != "" {
		if cachedResponse, found := p.cache.Get(promptText); found {
			log.Printf("⚡ [CoreRoute-AI Cache] HIT! Serving response in <5ms for prompt: '%s'", promptText)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-CoreRoute-Cache", "HIT")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(cachedResponse))
			return
		}
	}

	log.Printf("miss [CoreRoute-AI Cache] MISS. Executing downstream inference...")

	// 2. CACHE MISS: Intercept response to save it later
	// We use a custom ResponseRecorder wrapper to capture the response bytes
	recorder := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           bytes.NewBuffer(nil),
	}

	// 3. ATTEMPT 1: Try Local Ollama First
	success := p.forwardToOllama(recorder, r, bodyBytes)
	if !success {
		// 4. FALLBACK TRIGGERED: Ollama failed, try Groq
		log.Printf("⚠️ [CoreRoute-AI] Primary Ollama node failed. Triggering cloud fallback to Groq...")
		p.forwardToGroq(recorder, r, bodyBytes)
	}

	// 5. If the request was successful, store the response in our semantic cache
	if recorder.statusCode == http.StatusOK && promptText != "" {
		p.cache.Set(promptText, recorder.body.String())
	}
}

// responseRecorder captures the HTTP response so we can cache it
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (rec *responseRecorder) WriteHeader(statusCode int) {
	rec.statusCode = statusCode
	rec.ResponseWriter.WriteHeader(statusCode)
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	rec.body.Write(b) // Save a copy to our buffer
	return rec.ResponseWriter.Write(b) // Write to the actual client
}

func (p *ResilientProxy) forwardToOllama(w http.ResponseWriter, r *http.Request, bodyBytes []byte) bool {
	targetURL := p.ollamaURL + r.URL.Path

	req, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return false
	}
	req.Header = r.Header.Clone()

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Printf("❌ Ollama connection error: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		log.Printf("❌ Ollama returned server error status: %d", resp.StatusCode)
		return false
	}

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	return true
}

func (p *ResilientProxy) forwardToGroq(w http.ResponseWriter, r *http.Request, bodyBytes []byte) {
	targetURL := p.groqURL + "/chat/completions"

	req, err := http.NewRequest(r.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(w, "Failed to build Groq fallback request", http.StatusInternalServerError)
		return
	}

	req.Header = r.Header.Clone()
	req.Header.Set("Authorization", "Bearer "+p.groqAPIKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Printf("❌ Groq fallback connection error: %v", err)
		http.Error(w, "Both local Ollama and cloud fallback failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	log.Printf("✅ Groq fallback successfully served request. Status: %s", resp.Status)

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}