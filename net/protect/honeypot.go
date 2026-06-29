package protect

import (
	"strings"

	"backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// defaultHoneypotPaths are common attack paths that are unlikely to be legitimate.
var defaultHoneypotPaths = []string{
	"/wp-admin",
	"/wp-login.php",
	"/phpmyadmin",
	"/myadmin",
	"/mysql",
	"/admin",
	"/administrator",
	"/shell",
	"/cgi-bin",
	"/.env",
	"/.git/config",
	"/actuator",
	"/health",
	"/metrics",
}

// HoneypotConfig holds configuration for the honeypot.
type HoneypotConfig struct {
	Enabled     bool
	Paths       []string
	LogOnly     bool // if true, only log, do not block
	BlockStatus int  // HTTP status to return when blocked (default 404)
}

// DefaultHoneypotConfig returns a production-friendly default.
func DefaultHoneypotConfig() HoneypotConfig {
	return HoneypotConfig{
		Enabled:     true,
		Paths:       defaultHoneypotPaths,
		LogOnly:     false,
		BlockStatus: 404,
	}
}

// Honeypot returns a middleware that traps malicious scanners.
func Honeypot(cfg HoneypotConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) { c.Next() }
	}
	pathsMap := make(map[string]bool)
	for _, p := range cfg.Paths {
		pathsMap[p] = true
	}
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// Exact match or prefix match (e.g., /wp-admin/anything)
		isHoneypot := false
		if pathsMap[path] {
			isHoneypot = true
		} else {
			for p := range pathsMap {
				if strings.HasPrefix(path, p+"/") {
					isHoneypot = true
					break
				}
			}
		}
		if isHoneypot {
			logger.Warn("honeypot triggered",
				"ip", c.ClientIP(),
				"path", path,
				"method", c.Request.Method,
				"user_agent", c.Request.UserAgent(),
			)
			if !cfg.LogOnly {
				c.AbortWithStatus(cfg.BlockStatus)
				return
			}
		}
		c.Next()
	}
}
