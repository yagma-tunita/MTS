// Package middleware 提供 HTTP 中间件，按执行顺序排列：
//   Logger → Recovery → CORS → SecurityHeaders → Honeypot → IPBlocklist → RequestGuard → RateLimit
// 每个中间件只负责一个关注点。
package middleware

import (
	"strings"

	"backend/pkg/errors"
	"backend/pkg/jwt"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT 认证鉴权中间件。
//
// 职责：
//   - 从 Authorization header 提取 Bearer token。
//   - 调用 jwtService.ValidateToken 验证令牌有效性。
//   - 将验证通过后的用户信息（user_id, username, role）注入 Gin Context。
//   - 提供角色校验层（RequireRole），用于 API 级别的权限控制。
//
// 使用方式：
//
//	authMw := middleware.NewAuthMiddleware(jwtSvc)
//	r.Use(authMw.RequireAuth())         // 全局或路由组级别认证
//	r.Use(authMw.RequireRole("admin"))  // 特定路由需要 admin 角色
//
// 注意：RequireRole 必须放在 RequireAuth 之后使用，因为
// RequireRole 依赖 RequireAuth 注入到 Context 中的 role 字段。
type AuthMiddleware struct {
	jwtService jwt.JWTService
}

// NewAuthMiddleware 创建认证中间件实例。
// jwtSvc 由 main 函数在初始化时传入，与 JWT 配置绑定。
func NewAuthMiddleware(jwtSvc jwt.JWTService) *AuthMiddleware {
	return &AuthMiddleware{jwtService: jwtSvc}
}

// RequireAuth 返回一个 Gin 中间件，要求请求携带有效的 JWT Bearer token。
//
// 认证流程：
//  1. 检查 Authorization header 是否存在。
//  2. 验证格式是否为 "Bearer <token>"（大小写不敏感）。
//  3. 提取 token 部分，调用 jwtService.ValidateToken 验证。
//  4. 验证通过后，将 user_id, username, role 注入 Gin Context。
//
// 错误处理：
//   - 401: Authorization header 缺失。
//   - 401: 格式不正确（不是 Bearer token）。
//   - 401: Token 无效或已过期。
//
// 注入到 Context 中的值可在 handler 层通过 c.Get() 获取：
//
//	userID, _ := c.Get("user_id")
//	role, _ := c.Get("role")
//	if role == "shipper" && userID != req.ShipperCompanyID {
//	    return 403
//	}
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.ErrorWithCode(c.Writer, errors.CodeUnauthorized, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.ErrorWithCode(c.Writer, errors.CodeUnauthorized, "invalid authorization format")
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := m.jwtService.ValidateToken(token)
		if err != nil {
			response.ErrorWithCode(c.Writer, errors.CodeUnauthorized, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RequireRole 返回一个 Gin 中间件，要求当前用户具备指定的角色之一。
//
// 这是基于角色的访问控制（RBAC）的简单实现。
// 当前系统支持三种角色：shipper、shipping、admin。
//
// 参数 allowedRoles 是允许的角色列表，只要匹配任意一个即通过。
// 例如 RequireRole("admin") 只允许管理员访问。
//
// error 处理：
//   - 403: Context 中没有 role 信息（可能未经过 RequireAuth）。
//   - 403: role 类型错误。
//   - 403: 当前角色不在 allowedRoles 列表中。
//
// 使用示例（路由配置在 router.go 中）：
//
//	adminGroup := protected.Group("/admin")
//	adminGroup.Use(authMw.RequireRole("admin"))
//	{
//	    adminGroup.POST("/register", h.Admin.Create)
//	}
func (m *AuthMiddleware) RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleRaw, exists := c.Get("role")
		if !exists {
			response.ErrorWithCode(c.Writer, errors.CodeForbidden, "missing role information")
			c.Abort()
			return
		}
		role, ok := roleRaw.(string)
		if !ok {
			response.ErrorWithCode(c.Writer, errors.CodeForbidden, "invalid role type")
			c.Abort()
			return
		}
		for _, allowed := range allowedRoles {
			if role == allowed {
				c.Next()
				return
			}
		}
		response.ErrorWithCode(c.Writer, errors.CodeForbidden, "insufficient permissions")
		c.Abort()
	}
}
