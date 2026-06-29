package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
)

var defaultCache *cache.Cache

func init() {
	defaultCache = cache.New(5*time.Minute, 10*time.Minute)
}

// Set stores a value with a TTL. If ttl <= 0, uses the default expiration.
func Set(key string, value interface{}, ttl time.Duration) {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	defaultCache.Set(key, value, ttl)
}

// Get retrieves a value; returns (nil, false) if not found or expired.
func Get(key string) (interface{}, bool) {
	return defaultCache.Get(key)
}

// Delete removes a key.
func Delete(key string) {
	defaultCache.Delete(key)
}

// DeletePrefix removes all keys starting with a given prefix (inefficient but ok for small caches).
// If you have many keys, consider storing a list or using a more advanced approach.
func DeletePrefix(prefix string) {
	items := defaultCache.Items()
	for k := range items {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			defaultCache.Delete(k)
		}
	}
}

// Flush clears the entire cache.
func Flush() {
	defaultCache.Flush()
}
