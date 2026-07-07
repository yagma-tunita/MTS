package protect

import (
	"backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// IPBlocklistConfig IP 黑名单配置。
//
// 用途：手动封禁某个恶意 IP（如某个 IP 持续发起攻击时，运维
// 将其加入黑名单）。默认关闭，由运维按需启用。
//
// 注意：如果请求经过反向代理（Nginx），c.ClientIP() 返回的是
// 代理的 IP 而非真实客户端 IP。需要在 Gin 中配置 TrustedProxies。
type IPBlocklistConfig struct {
	Enabled   bool     // 是否启用黑名单
	Blocklist []string // IP 地址列表
	BlockMsg  string   // 拦截时返回的消息
}

// DefaultIPBlocklistConfig 返回空黑名单（默认不启用）。
//
// 默认不启用黑名单的原因是：在系统上线初期，还不知道哪些 IP
// 是恶意的。运维人员可以通过监控日志发现异常 IP 后，
// 动态配置黑名单。
func DefaultIPBlocklistConfig() IPBlocklistConfig {
	return IPBlocklistConfig{
		Enabled:   false,
		Blocklist: []string{},
		BlockMsg:  "access denied",
	}
}

// IPBlocklist 返回一个 IP 黑名单中间件。
//
// 实现方式：
//   - 将配置中的 IP 列表转换为 map[string]bool，实现 O(1) 查询。
//   - 每个请求检查客户端 IP 是否在 map 中。
//   - 命中时记录 warn 日志并返回 403。
//
// 注意：当前实现只支持精确的 IP 地址匹配，不支持 CIDR 网段
// （如 192.168.1.0/24）。如果需要网段匹配，可以扩展此函数。
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
