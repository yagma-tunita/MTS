package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSConfig 跨域资源共享配置。
//
// 控制浏览器端跨域请求的行为。前端如果部署在不同域名或端口下
// （如前端 localhost:3000，后端 localhost:8080），必须配置 CORS。
type CORSConfig struct {
	AllowOrigins     []string      // 允许的源（Origin），如 ["http://localhost:3000"]。生产环境应限制为具体域名。
	AllowMethods     []string      // 允许的 HTTP 方法，如 ["GET", "POST", "PUT", "DELETE", "OPTIONS"]。
	AllowHeaders     []string      // 允许的请求头，如 ["Origin", "Content-Type", "Authorization"]。
	ExposeHeaders    []string      // 允许浏览器访问的响应头，如 ["Content-Length"]。
	AllowCredentials bool          // 是否允许携带 Cookie。如果设为 true，AllowOrigins 不能包含 "*"。
	MaxAge           time.Duration // 预检请求（OPTIONS）的缓存时间，单位秒。缓存期间浏览器不再发送预检请求。
}

// DefaultCORSConfig 返回适合生产的默认跨域配置。
//
// 默认值：
//   - AllowOrigins: ["*"] — 允许所有源（生产环境应改为具体域名）。
//   - AllowMethods: GET, POST, PUT, DELETE, OPTIONS — 标准的 REST 方法。
//   - AllowHeaders: Origin, Content-Type, Authorization — 前端常用的请求头。
//   - AllowCredentials: false — 不携带 Cookie，因为 JWT 存在 Header 中。
//   - MaxAge: 12 小时 — 预检请求缓存时间，减少 OPTIONS 请求次数。
//
// 注意：当 AllowOrigins 包含 "*" 时，AllowCredentials 必须为 false，
// 否则浏览器会报错（这是 CORS 规范的安全限制）。
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}

// NewCORS 根据配置创建 Gin CORS 中间件。
//
// 使用 gin-contrib/cors 库实现，内部处理 OPTIONS 预检请求和
// 响应头的设置。
//
// 在中间件链中的位置：放在 Logger 和 Recovery 之后，
// 但在认证和安全中间件之前。因为 CORS 是 HTTP 协议层面的
// 处理，与业务逻辑无关。
func NewCORS(cfg CORSConfig) gin.HandlerFunc {
	corsCfg := cors.Config{
		AllowOrigins:     cfg.AllowOrigins,
		AllowMethods:     cfg.AllowMethods,
		AllowHeaders:     cfg.AllowHeaders,
		ExposeHeaders:    cfg.ExposeHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	}
	return cors.New(corsCfg)
}
