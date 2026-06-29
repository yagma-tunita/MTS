package middleware

import (
	"sync"

	"backend/pkg/errors"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterConfig holds settings.
type RateLimiterConfig struct {
	Rate  rate.Limit // tokens per second
	Burst int        // max burst size
}

// DefaultRateLimiterConfig returns a sensible default.
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		Rate:  100,
		Burst: 20,
	}
}

type ipRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.RWMutex
	cfg RateLimiterConfig
}

func newIPRateLimiter(cfg RateLimiterConfig) *ipRateLimiter {
	return &ipRateLimiter{
		ips: make(map[string]*rate.Limiter),
		cfg: cfg,
	}
}

func (l *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	limiter, exists := l.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(l.cfg.Rate, l.cfg.Burst)
		l.ips[ip] = limiter
	}
	return limiter
}

// RateLimit returns middleware with configurable limiter.
func RateLimit(cfg RateLimiterConfig) gin.HandlerFunc {
	limiter := newIPRateLimiter(cfg)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.getLimiter(ip).Allow() {
			response.ErrorWithCode(c.Writer, errors.CodeTooManyRequests, "too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}
