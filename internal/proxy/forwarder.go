package proxy

import (
	"bytes"
	"io"
	"log"
	"net/http"
)

type ResilientProxy struct {
	ollamaURL  string
	groqURL    string
	groqAPIKey string
	httpClient *http.Client
}

func NewResilientProxy(ollamaURL, groqAPIKey string) *ResilientProxy {
	return &ResilientProxy{
		ollamaURL:  ollamaURL,
		groqURL:    "https://api.groq.com/openai/v1",
		groqAPIKey: groqAPIKey,
		httpClient: &http.Client{},
	}
}

func (p *ResilientProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	success := p.forwardToOllama(w, r, bodyBytes)
	if success {
		return 
	}

	log.Printf("⚠️ [CoreRoute-AI] Primary Ollama node failed. Triggering cloud fallback to Groq...")
	p.forwardToGroq(w, r, bodyBytes)
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