// Package websocket 提供 WebSocket 服务，支持实时消息推送。
//
// 架构：Hub-Client 模式。
//
//   Hub（中央调度器）
//   ├── register   chan ← 新连接注册
//   ├── unregister chan ← 连接断开注销
//   ├── broadcast  chan ← 广播消息
//   └── clients    map[*Client]bool ← 所有活跃连接
//
//   Client A ──→ ReadPump (goroutine) ← 读取客户端消息
//           ──→ WritePump (goroutine) ← 向客户端写消息
//           ──→ send chan ← 接收待发送消息的缓冲区
//
// 为什么用两个 goroutine（ReadPump / WritePump）而不是一个：
//   gorilla/websocket 的底层 TCP 连接是全双工的，可以同时读写。
//   但 gorilla/websocket 的 Conn 对象不是 goroutine 安全的——
//   并发读写同一个 Conn 会导致竞态。标准的做法是用两个 goroutine，
//   一个只读、一个只写，通过 channel 传递消息。
//
// 消息流转路径：
//   业务层调用 PushToUser(userID, role, message)
//   → hub.SendToUser() 遍历 clients map
//   → 匹配 userID+role 的 client.send channel
//   → WritePump 从 channel 接收到消息
//   → conn.WriteMessage() 发送 WebSocket 数据帧
//
// 当前用途：
//   订单状态变更通知。当 PUT /orders/{id}/status 更新状态时，
//   通过 WebSocket 向该订单的货主推送状态更新。
package websocket

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// writeWait 写入超时时间。
	// WritePump 在 conn.SetWriteDeadline 中使用此值。
	// 如果 10 秒内无法写入数据（如网络拥堵），写操作超时，
	// WritePump 退出，连接被关闭。
	writeWait = 10 * time.Second

	// pongWait 等待 Pong 响应的超时时间。
	// ReadPump 在每次收到 Pong 后重置读取截止时间为 now + pongWait。
	// 如果超过 pongWait 没有收到任何消息（包括 Pong），
	// 读取超时，ReadPump 退出，连接被关闭。
	pongWait = 60 * time.Second

	// pingPeriod 发送 Ping 的间隔。
	// 设为 pongWait 的 90%（54 秒），留 6 秒的余量覆盖网络延迟。
	// 如果设为 60 秒，恰好在一个 pingWait 周期结束前发 ping，
	// 网络延迟可能导致 pong 回来时已经超时。
	pingPeriod = (pongWait * 9) / 10

	// maxMsgSize 允许接收的最大消息大小（字节）。
	// 当前系统不处理客户端发来的消息，但为了防止恶意客户端
	// 发送巨大消息消耗内存，设置一个较小的上限。
	maxMsgSize = 512
)

// Client 表示一个 WebSocket 客户端连接。
//
// 每个 Client 对应一个 HTTP 连接升级后的 WebSocket 连接。
// 生命周期：
//   NewClient → hub.register → go WritePump → go ReadPump
//   → ReadPump 检测到断开 → hub.unregister → 连接关闭
//
// 字段说明：
//   - hub: 所属的 Hub，用于在断开时发送 unregister 信号。
//   - conn: 底层 WebSocket 连接。
//   - send: 缓冲 channel，容量 256 条消息。
//     如果消息生产速度大于消费速度，超过 256 条时新消息被丢弃
//     （select default 分支），而不是阻塞生产者。
//   - userID + role: 用户的身份标识，用于按用户推送消息。
//     hub.SendToUser 通过这两个字段匹配目标客户端。
//   - active: 原子操作的活跃标志，用于快速判断连接是否已关闭。
type Client struct {
	hub      *Hub           // 所属 Hub 实例
	conn     *websocket.Conn // WebSocket 连接
	send     chan []byte     // 待发送消息缓冲区
	userID   int64          // 用户 ID（公司/管理员 ID）
	role     string         // 用户角色（shipper/shipping/admin）
	idString string         // "userID:role" 格式，用于日志
	active   int32          // 1=活跃，0=已关闭（atomic 读写）
}

// NewClient 创建一个新的 WebSocket 客户端连接。
//
// 参数：
//   - hub: 管理此连接的 Hub。
//   - conn: 已升级的 WebSocket 连接。
//   - userID: 用户 ID（从 JWT 中解析）。
//   - role: 用户角色（从 JWT 中解析）。
func NewClient(hub *Hub, conn *websocket.Conn, userID int64, role string) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		role:     role,
		idString: fmt.Sprintf("%d:%s", userID, role),
		active:   1,
	}
}

// ReadPump 从 WebSocket 连接读取消息的 goroutine。
//
// 当前实现只读取并丢弃消息（_ 忽略消息内容），主要用于：
//   1. 检测连接是否断开（ReadMessage 在断开时返回 error）。
//   2. 处理 Pong 响应（更新读取截止时间）。
//   3. 维持读取超时机制。
//
// 连接关闭流程：
//   1. ReadMessage 返回 error（连接断开或读取超时）。
//   2. 将 active 标志设为 0（标记连接已关闭）。
//   3. 将自身发送到 hub.unregister 通道（从 Hub 中移除）。
//   4. 关闭底层连接。
//
// 为什么不处理客户端发来的消息：
//   当前系统的 WebSocket 只用于服务端推送（推送订单状态变更），
//   不需要客户端发送消息。如果将来需要实现双向通信
//   （如客户端发送聊天消息），可以在此 goroutine 中解析消息。
func (c *Client) ReadPump() {
	defer func() {
		atomic.StoreInt32(&c.active, 0)
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// WritePump 将消息写入 WebSocket 连接的 goroutine。
//
// 从 send channel 接收消息并写入 WebSocket 连接。
// 同时定期发送 Ping 帧保活。
//
// 为什么需要定期 Ping：
//   中间的网络设备（负载均衡器、防火墙、NAT 路由器）可能因为
//   长时间没有数据包而断开 TCP 连接（超时断连）。
//   Ping 帧维持连接活跃，防止被中间设备切断。
//
// 退出条件：
//   - send channel 被关闭：Service 层停止推送（连接被关闭）。
//   - ping 或写操作失败：网络断开或连接已关闭。
//     注意：WebSocket 发送 Ping 失败不一定意味着连接已断开，
//     但为了简单处理，任何写操作失败都直接退出。
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// send channel 已关闭（Hub 移除 Client 时关闭），
				// 发送 Close 消息后退出
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// IsActive 返回连接是否仍然活跃。
//
// 使用 atomic.LoadInt32 确保并发安全。
// ReadPump 会在检测到断开时第一时间将 active 设为 0，
// hub.SendToUser 通过此方法过滤已断开的连接，避免
// 向已关闭的 send channel 发送消息（会 panic）。
func (c *Client) IsActive() bool {
	return atomic.LoadInt32(&c.active) == 1
}
