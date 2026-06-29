package websocket

import (
	"net/http"

	"backend/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, restrict to your domain
		return true
	},
}

var hub = NewHub()

func init() {
	go hub.Run()
}

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

// PushToUser sends message to a specific user's all connections.
func PushToUser(userID int64, role string, message []byte) {
	hub.SendToUser(userID, role, message)
}
