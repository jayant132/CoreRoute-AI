package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func NewOllamaProxy(ollamaHost string) http.Handler {

	target, err := url.Parse(ollamaHost)
	if err != nil {
		log.Fatalf("Invalid Ollama URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		
		req.Host = target.Host
		
		log.Printf("[CoreRoute-AI Proxy] Forwarding request to Ollama: %s %s", req.Method, req.URL.Path)
	}

	return proxy
}