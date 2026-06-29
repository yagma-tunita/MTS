package protect

import (
	"backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// IPBlocklistConfig holds configuration.
type IPBlocklistConfig struct {
	Enabled   bool
	Blocklist []string
	BlockMsg  string
}

// DefaultIPBlocklistConfig returns empty blocklist.
func DefaultIPBlocklistConfig() IPBlocklistConfig {
	return IPBlocklistConfig{
		Enabled:   false,
		Blocklist: []string{},
		BlockMsg:  "access denied",
	}
}

// IPBlocklist returns a middleware that rejects IPs in the blocklist.
func IPBlocklist(cfg IPBlocklistConfig) gin.HandlerFunc {
	if !cfg.Enabled || len(cfg.Blocklist) == 0 {
		return func(c *gin.Context) { c.Next() }
	}
	blockMap := make(map[string]bool)
	for _, ip := range cfg.Blocklist {
		blockMap[ip] = true
	}
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if blockMap[clientIP] {
			logger.Warn("blocked IP", "ip", clientIP, "path", c.Request.URL.Path)
			c.AbortWithStatusJSON(403, gin.H{"code": 403, "message": cfg.BlockMsg})
			return
		}
		c.Next()
	}
}
