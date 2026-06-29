package middleware

import (
	"strings"

	"backend/pkg/errors"
	"backend/pkg/jwt"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	jwtService jwt.JWTService
}

func NewAuthMiddleware(jwtSvc jwt.JWTService) *AuthMiddleware {
	return &AuthMiddleware{jwtService: jwtSvc}
}

// RequireAuth validates JWT and sets user context.
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

// RequireRole adds role-based authorization.
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
