package middleware

import (
	"fmt"
	"runtime/debug"

	"backend/pkg/errors"
	"backend/pkg/logger"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// Recovery returns a middleware that recovers from panics and returns 500.
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
