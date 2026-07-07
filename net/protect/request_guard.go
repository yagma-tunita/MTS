package protect

import (
	"net/http"
	"strings"

	"backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

// RequestGuardConfig 请求安全防护配置。
//
// 执行四个维度的安全检查：
//   1. HTTP 方法白名单：只允许 GET/POST/PUT/DELETE/PATCH/OPTIONS。
//   2. URL 长度限制：防止超长 URL（可能用于缓冲区溢出攻击）。
//   3. 请求体大小限制：防止大请求体耗尽内存。
//   4. User-Agent 黑名单：拦截已知的恶意工具。
type RequestGuardConfig struct {
	Enabled           bool     // 是否启用
	MaxBodyBytes      int64    // 最大请求体大小（字节），0=不限制。默认 4MB。
	MaxURLLength      int      // 最大 URL 长度（字符数），0=不限制。默认 2048。
	AllowedMethods    []string // 允许的 HTTP 方法列表，空=不限制。
	BlockedUserAgents []string // 要拦截的 User-Agent 子串（大小写不敏感匹配）。
	BlockMessage      string   // 拦截时返回的消息。
}

// DefaultRequestGuardConfig 返回默认请求防护配置。
//
// 配置项选择理由：
//   - MaxBodyBytes=4MB：正常的 API 请求体不会超过 4MB。
//     Excel 导入文件虽然在 4MB 以上，但那是 multipart/form-data，
//     Gin 的分包解析会在文件上传时单独处理大文件。
//   - MaxURLLength=2048：绝大多数请求 URL 不会超过 2048 字符。
//     超过这个长度的一般是攻击请求或配置错误的客户端。
//   - AllowedMethods：只允许 RESTful 的常见方法。
//     TRACE/CONNECT 等方法通常用于攻击。
//   - BlockedUserAgents：默认为空，由运维根据攻击日志补充。
func DefaultRequestGuardConfig() RequestGuardConfig {
	return RequestGuardConfig{
		Enabled:           true,
		MaxBodyBytes:      4 * 1024 * 1024,
		MaxURLLength:      2048,
		AllowedMethods:    []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions},
		BlockedUserAgents: []string{},
		BlockMessage:      "request rejected by security policy",
	}
}

// RequestGuard 返回一个请求安全守卫中间件。
//
// 执行顺序：
//  1. HTTP 方法白名单校验 —— 如果方法不在允许列表中，返回 405。
//     注意：OPTIONS 在列表中，所以 CORS 预检请求不会受影响。
//  2. URL 长度限制 —— 如果 URL 超长，返回 414 URI Too Long。
//  3. 请求体大小限制 —— 通过 http.MaxBytesReader 包装请求体，
//     当读取 Body 超过限制时自动返回 413 Payload Too Large。
//     注意：MaxBytesReader 是延迟校验的，只在读取 Body 时触发。
//     如果 handler 不读取 Body，限制不会生效。
//  4. User-Agent 黑名单 —— 子串匹配，大小写不敏感。
//     如果命中，返回 403。
//
// 关于 MaxBytesReader 的工作原理：
//   http.MaxBytesReader 返回一个 Reader，当读取的累计字节数
//   超过限制时，后续 Read 调用返回 error（"http: request body too large"），
//   Gin 的 c.ShouldBindJSON 读取 Body 时遇到此 error 会返回 400。
//   不需要额外写逻辑来处理 body 过大。
//
// 为什么允许的方法中包含 OPTIONS：
//   浏览器的 CORS 预检请求使用 OPTIONS 方法。如果不允许 OPTIONS，
//   前端跨域请求会在预检阶段就被拦截，后续的 GET/POST 请求发不出去。
func RequestGuard(cfg RequestGuardConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) { c.Next() }
	}

	allowedMethods := make(map[string]bool)
	for _, m := range cfg.AllowedMethods {
		allowedMethods[strings.ToUpper(m)] = true
	}

	blockedUAs := make([]string, len(cfg.BlockedUserAgents))
	for i, ua := range cfg.BlockedUserAgents {
		blockedUAs[i] = strings.ToLower(ua)
	}

	return func(c *gin.Context) {
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

		if cfg.MaxBodyBytes > 0 {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, cfg.MaxBodyBytes)
		}

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
