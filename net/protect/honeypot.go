// Package protect 提供 HTTP 安全防护中间件。
// 包含四个独立的中间件，每个处理一个安全关注点。
package protect

import (
	"strings"

	"backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// defaultHoneypotPaths 是蜜罐陷阱的默认监控路径列表。
//
// 这些是常见扫描器和自动化攻击工具默认会探测的路径：
//   - /wp-admin, /wp-login.php — WordPress 管理后台，全网扫描最多
//   - /phpmyadmin, /myadmin — 数据库管理工具
//   - /mysql — MySQL 默认路径
//   - /admin, /administrator — 通用管理后台
//   - /shell — WebShell 后门
//   - /cgi-bin — CGI 漏洞扫描
//   - /.env — 环境变量泄露
//   - /.git/config — Git 信息泄露
//   - /actuator, /metrics — Spring Boot Actuator（微服务框架端点）
//
// 正常的 MTS 用户不会访问这些路径。
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
	"/metrics",
}

// HoneypotConfig 蜜罐中间件配置。
//
// LogOnly 模式：只记录日志不拦截，用于初次部署时评估误报率。
// 监控几天确认没有合法请求触发后，再关闭 LogOnly 开始拦截。
type HoneypotConfig struct {
	Enabled     bool     // 是否启用蜜罐
	Paths       []string // 蜜罐路径列表
	LogOnly     bool     // true=仅记录日志，不拦截（观察模式）
	BlockStatus int      // 拦截时返回的 HTTP 状态码（默认 404，迷惑攻击者）
}

// DefaultHoneypotConfig 返回默认蜜罐配置：启用，拦截，返回 404。
//
// 为什么拦截时返回 404 而不是 403：
//   403 是"服务器拒绝了请求"，攻击者知道服务器存在但拒绝访问。
//   404 是"路径不存在"，攻击者会认为扫描到的是无效路径，
//   不会继续对这个"不存在的路径"深入攻击。
func DefaultHoneypotConfig() HoneypotConfig {
	return HoneypotConfig{
		Enabled:     true,
		Paths:       defaultHoneypotPaths,
		LogOnly:     false,
		BlockStatus: 404,
	}
}

// Honeypot 返回一个蜜罐中间件。
//
// 匹配规则：
//   - 精确匹配：路径完全等于列表中的某个路径。
//   - 前缀匹配：路径以 "列表路径 + /" 开头（如 /wp-admin/anything）。
//
// 为什么需要前缀匹配：
//   攻击者访问 /wp-admin/plugins.php 时，也应当被拦截。
//   如果只做精确匹配，/wp-admin 被匹配但不拦截 /wp-admin/anything。
//
// 当命中蜜罐路径时：
//   - 以 warn 级别记录日志（包含 IP、路径、方法、User-Agent）。
//   - 如果不是 LogOnly 模式，返回配置的状态码并终止请求。
//   - 注意不会记录 request_id，因为蜜罐请求通常没有经过 Logger 中间件
//     的完整处理链。但 Logger 中间件位于蜜罐之前，所以 request_id 已经生成。
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
