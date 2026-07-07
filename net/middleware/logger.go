package middleware

import (
	"time"

	"backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Logger 返回一个记录 HTTP 请求日志的中间件。
//
// 功能：
//   - 为每个请求自动生成或透传 request_id，用于链路追踪。
//   - 记录请求的方法、路径、状态码、耗时、客户端 IP 和 User-Agent。
//
// request_id 生成规则：
//   - 优先读取请求头 X-Request-ID（如果调用方传了，说明已经在
//     其他服务中生成，透传下去做全链路追踪）。
//   - 如果请求头为空，使用 uuid.New() 生成一个。
//   - 设置的 response header X-Request-ID 中，方便前端或下游服务获取。
//
// 日志格式（JSON 模式）：
//
//	{"time":"...","level":"INFO","msg":"HTTP request",
//	 "request_id":"abc-123","method":"GET","path":"/api/v1/orders/1",
//	 "status":200,"latency_ms":42,"ip":"192.168.1.1","user_agent":"curl/7.68"}
//
// 为什么放在中间件链的最前面：
//   只有放在最前面，才能记录到所有请求——包括认证失败、参数错误
//   被后续中间件拦截的请求。如果放在后面，一些请求在被拒绝前
//   就已经被 logger 记录到了。
//
// 为什么记录 latency_ms 而不是 latency：
//   毫秒单位在日志分析中更直观，Elasticsearch 中对 latency_ms
//   做聚合计算也方便。
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()

		logger.Info("HTTP request",
			"request_id", requestID,
			"method", method,
			"path", path,
			"status", statusCode,
			"latency_ms", latency.Milliseconds(),
			"ip", clientIP,
			"user_agent", c.Request.UserAgent(),
		)
	}
}
