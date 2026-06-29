package protect

import (
	"net/http"
	"strings"

	"backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RequestGuardConfig holds security limits for HTTP requests.
type RequestGuardConfig struct {
	Enabled           bool
	MaxBodyBytes      int64    // maximum request body size, 0 = no limit
	MaxURLLength      int      // maximum URL length in characters, 0 = no limit
	AllowedMethods    []string // if non-empty, only these methods are allowed
	BlockedUserAgents []string // substrings to match (case-insensitive)
	BlockMessage      string   // response message when blocked
}

// DefaultRequestGuardConfig returns a production-friendly default.
func DefaultRequestGuardConfig() RequestGuardConfig {
	return RequestGuardConfig{
		Enabled:           true,
		MaxBodyBytes:      4 * 1024 * 1024, // 4 MB
		MaxURLLength:      2048,
		AllowedMethods:    []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions},
		BlockedUserAgents: []string{},
		BlockMessage:      "request rejected by security policy",
	}
}

// RequestGuard returns a middleware that enforces security limits.
func RequestGuard(cfg RequestGuardConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) { c.Next() }
	}

	// Precompute allowed methods map
	allowedMethods := make(map[string]bool)
	for _, m := range cfg.AllowedMethods {
		allowedMethods[strings.ToUpper(m)] = true
	}

	// Precompile blocked user-agents to lower case for case-insensitive match
	blockedUAs := make([]string, len(cfg.BlockedUserAgents))
	for i, ua := range cfg.BlockedUserAgents {
		blockedUAs[i] = strings.ToLower(ua)
	}

	return func(c *gin.Context) {
		// 1. Enforce allowed methods
		if len(allowedMethods) > 0 && !allowedMethods[c.Request.Method] {
			logger.Warn("request guard: method not allowed",
				"method", c.Request.Method,
				"ip", c.ClientIP(),
				"path", c.Request.URL.Path,
			)
			c.AbortWithStatusJSON(http.StatusMethodNotAllowed, gin.H{
				"code":    http.StatusMethodNotAllowed,
				"message": cfg.BlockMessage,
			})
			return
		}

		// 2. Enforce max URL length
		if cfg.MaxURLLength > 0 && len(c.Request.URL.String()) > cfg.MaxURLLength {
			logger.Warn("request guard: URL too long",
				"length", len(c.Request.URL.String()),
				"ip", c.ClientIP(),
				"path", c.Request.URL.Path,
			)
			c.AbortWithStatusJSON(http.StatusRequestURITooLong, gin.H{
				"code":    http.StatusRequestURITooLong,
				"message": cfg.BlockMessage,
			})
			return
		}

		// 3. Enforce max body size (limit the request body reader)
		if cfg.MaxBodyBytes > 0 {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, cfg.MaxBodyBytes)
		}

		// 4. Block specific user-agents (simple substring match)
		if len(blockedUAs) > 0 {
			userAgent := strings.ToLower(c.Request.UserAgent())
			for _, blocked := range blockedUAs {
				if strings.Contains(userAgent, blocked) {
					logger.Warn("request guard: blocked user-agent",
						"user_agent", c.Request.UserAgent(),
						"ip", c.ClientIP(),
						"path", c.Request.URL.Path,
					)
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"code":    http.StatusForbidden,
						"message": cfg.BlockMessage,
					})
					return
				}
			}
		}

		c.Next()
	}
}
