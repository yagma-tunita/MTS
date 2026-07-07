package websocket

import (
	"net/http"

	"backend/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// upgrader 将 HTTP 连接升级为 WebSocket。生产环境应限制 CheckOrigin 为具体域名。
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// hub 全局 WebSocket Hub 单例（包级变量，避免穿透多层参数传递）
var hub = NewHub()

// init 包初始化时启动 Hub 事件循环 goroutine。
func init() { go hub.Run() }

// ServeWS 处理 /ws 升级请求。JWT 认证通过 URL query token 传参
// （浏览器 WebSocket API 不支持自定义 Header）
func ServeWS(jwtSvc jwt.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims, err := jwtSvc.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		client := NewClient(hub, conn, claims.UserID, claims.Role)
		hub.register <- client
		go client.WritePump()
		go client.ReadPump()
	}
}

// PushToUser 业务层入口：向指定 userID+role 推送消息。线程安全。
func PushToUser(userID int64, role string, message []byte) { hub.SendToUser(userID, role, message) }

// ShutdownHub 优雅关闭 Hub（在 srv.Shutdown 之前调用，先关 WS 再关 HTTP）
func ShutdownHub() { hub.Stop() }
