package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// CacheItem holds the cached response and its timestamp
type CacheItem struct {
	Response  string
	Timestamp time.Time
}

// SemanticCache is a thread-safe in-memory cache for LLM responses
type SemanticCache struct {
	mu    sync.RWMutex
	items map[string]CacheItem
}

func NewSemanticCache() *SemanticCache {
	return &SemanticCache{
		items: make(map[string]CacheItem),
	}
}

// hashPrompt normalizes a prompt string and generates a secure hash key
func hashPrompt(prompt string) string {
	cleaned := strings.ToLower(strings.TrimSpace(prompt))
	hash := sha256.Sum256([]byte(cleaned))
	return hex.EncodeToString(hash[:])
}

// Get checks if a semantically similar prompt exists in the cache
func (c *SemanticCache) Get(prompt string) (string, bool) {
	key := hashPrompt(prompt)

	c.mu.RLock() // <-- FIXED: Added parentheses here!
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return "", false
	}

	// Items expire after 1 hour
	if time.Since(item.Timestamp) > 1*time.Hour {
		return "", false
	}

	return item.Response, true
}

// Set stores a prompt response into the cache safely
func (c *SemanticCache) Set(prompt string, response string) {
	key := hashPrompt(prompt)

	c.mu.Lock() 
	defer c.mu.Unlock()

	c.items[key] = CacheItem{
		Response:  response,
		Timestamp: time.Now(),
	}
}