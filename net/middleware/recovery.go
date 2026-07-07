package middleware

import (
	"fmt"
	"runtime/debug"

	"backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// Recovery 返回一个异常恢复中间件，捕获 handler 中的 panic。
//
// panic 来源：
//   - handler 中未处理的运行时错误（如空指针解引用、数组越界）。
//   - 业务代码中直接调用的 panic（例如 "panic("should not happen")"）。
//   - 第三方库的内部 panic。
//
// 处理流程：
//  1. defer 中通过 recover() 捕获 panic。
//  2. 使用 debug.Stack() 获取完整调用堆栈。
//  3. 以 error 级别记录日志（包含堆栈信息，便于定位问题）。
//  4. 向客户端返回 500 Internal Server Error。
//  5. 调用 c.Abort() 终止请求链。
//
// 为什么不在中间件中恢复执行而是直接返回 500：
//   panic 意味着发生了代码逻辑错误或无法恢复的异常。
//   此时继续执行 handler 可能产生更严重的后果（如数据不一致）。
//   最安全的做法是立即停止当前请求的处理，返回 500，
//   让服务的外部监控（如健康检查）检测到异常。
//
// 日志中的 request_id 可用于关联同一请求的其他日志条目，
// 方便排查具体是哪个请求触发了 panic。
//
// 为什么 Recovery 放在 Logger 之后、其他中间件之前：
//   如果放在 Logger 之前，panic 发生时 logger 无法记录请求信息。
//   如果放在其他中间件之后，那些中间件内的 panic 无法被 Recovery 捕获。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				logger.Error("panic recovered",
					"error", fmt.Sprintf("%v", err),
					"stack", string(stack),
					"request_id", c.GetString("request_id"),
				)
				response.ErrorWithCode(c.Writer, errors.CodeInternal, "internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}
