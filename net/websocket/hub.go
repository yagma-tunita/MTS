package websocket

import (
	"sync"
)

// Hub 管理所有 WebSocket 客户端连接。
//
// 设计模式：Actor 模式（事件循环 + channel 驱动）。
// 所有对 clients map 的访问都集中在 Run() 的 select 中串行处理，
// 避免了并发 map 操作需要加锁的问题。
//
// 字段说明：
//   clients    — 所有活跃连接的集合。key=*Client, value=true。
//                Client 断开时从此 map 中移除。
//   register   — 连接注册 channel。ServeWS 创建新 Client 后，
//                将 Client 发送到此 channel，Run() 将其加入 clients。
//   unregister — 连接注销 channel。Client.ReadPump 检测到断开后，
//                将自身发送到此 channel，Run() 将其从 clients 移除
//                并关闭 send channel 触发 WritePump 退出。
//   broadcast  — 全局广播 channel（当前未使用，预留扩展）。
//                向所有客户端发送相同消息。
//   stop       — 停止信号。ShutdownHub 调用 Stop() 时关闭此 channel，
//                Run() 收到信号后遍历所有连接逐个关闭并退出。
//   mu         — 读写锁。保护 SendToUser 中对 clients 的遍历。
//                使用 RWMutex 的读锁允许多个推送 goroutine 同时执行。
//   stopOnce   — 确保 Stop() 只执行一次。防止多次 close(stop) 导致 panic。
//
// 消息分发方式：两种模式
//   1. 按用户推送（SendToUser）：使用 RWMutex + 遍历 clients map。
//      这种模式适合"精确推送"——只推送给特定 userID+role 的用户。
//   2. 全局广播（broadcast channel）：通过事件循环串行处理。
//      这种模式适合"广播"——向所有在线用户推送同一条消息。
//
// 为什么不统一用 channel 方式：
//   按用户推送是高频操作（订单状态变更时触发）。
//   如果每个推送请求都通过 channel 发送到事件循环再处理，
//   事件循环会成为瓶颈。RWMutex 允许多个推送并发执行。
//
// 为什么不统一用 RWMutex + 遍历：
//   全局广播发生时，如果使用 RWMutex 加锁遍历，
//   推送线程会长时间持有读锁（遍历所有客户端），
//   阻塞连接注册/注销等写操作。channel 方式更优雅。
type Hub struct {
	clients    map[*Client]bool // 所有已注册的客户端（key=Client指针，value=true）
	register   chan *Client     // 注册通道（新 WebSocket 连接建立时发送）
	unregister chan *Client     // 注销通道（连接断开时发送）
	broadcast  chan []byte      // 广播通道（向所有客户端发送消息，当前未使用）
	stop       chan struct{}    // 停止信号（服务关闭时触发）
	mu         sync.RWMutex     // 保护 SendToUser 对 clients 的并发遍历
	stopOnce   sync.Once        // 确保 Stop 只执行一次，防止重复 close(channel)
}

// NewHub 创建并返回一个新的 Hub 实例。
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		stop:       make(chan struct{}),
	}
}

// Run 启动 Hub 的事件循环。
//
// 在一个独立的 goroutine 中运行（由 init() 在包加载时启动）。
// 处理四种事件（通过 select 多路复用）：
//
// register:
//   将 Client 加入 clients map，标记为活跃。之后该客户端可以接收推送消息。
//   注意：注册时不需要启动 goroutine（ReadPump 和 WritePump 在 ServeWS 中已启动）。
//
// unregister:
//   从 clients map 移除 Client，然后 close(client.send)。
//   关闭 send channel 会使 WritePump 的 <-c.send 收到零值 + ok=false，
//   WritePump 据此判断连接已关闭，退出 goroutine 并关闭底层 conn。
//
// broadcast:
//   遍历所有客户端，将消息发送到 client.send channel。
//   使用 select + default 实现非阻塞发送：如果某个客户端的 send buffer
//   已满（消费速度慢于生产速度），跳过该客户端，不阻塞其他客户端的发送。
//
// stop:
//   立即关闭所有连接：遍历 clients map，逐个 close(send) 和 conn.Close()。
//   清空 map 后退出事件循环。
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send) // 关闭 send channel 触发 WritePump 退出
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// send channel 缓冲区满时跳过，不阻塞当前 goroutine
				}
			}
			h.mu.RUnlock()

		case <-h.stop:
			h.mu.Lock()
			for client := range h.clients {
				delete(h.clients, client)
				close(client.send)
				client.conn.Close()
			}
			h.mu.Unlock()
			return
		}
	}
}

// Stop 发送停止信号，优雅关闭 Hub。
//
// 使用 sync.Once 确保关闭动作只执行一次。
// 多次调用 Stop 不会 panic（close 已关闭的 channel 会导致 panic）。
// 调用 Stop 后 Hub.Run() 会关闭所有连接并退出。
func (h *Hub) Stop() {
	h.stopOnce.Do(func() {
		close(h.stop)
	})
}

// SendToUser 向指定用户（userID + role 匹配）的所有活跃连接发送消息。
//
// 参数：
//   - userID: 目标用户 ID（货主公司 ID / 船公司 ID / 管理员 ID）。
//   - role: 目标用户角色（"shipper" / "shipping" / "admin"）。
//   - message: 已序列化的 JSON 字节数据。
//
// 匹配规则：
//   遍历所有客户端，筛选出 client.userID == userID && client.role == role
//   且 IsActive() 返回 true 的连接。向每个匹配连接的 send channel 发送消息。
//
// 为什么不阻塞：
//   使用 select + default 实现非阻塞发送。如果某个连接的 send buffer 已满
//   （说明客户端消费速度慢或已断开但尚未执行 unregister），跳过该连接。
//   这是"尽力投递"语义——推送消息是实时通知，错过了一条用户通常不会察觉。
//   如果阻塞等待，会导致所有消息推送延迟，影响其他正常用户的体验。
//
// 为什么检查 IsActive：
//   在 ReadPump 检测到断开（ReadMessage 返回 error）到实际执行
//   unregister（向 hub.unregister channel 发送信号）之间有一个时间窗口。
//   在此期间，Client 仍在 clients map 中，但底层连接已经断开。
//   IsActive 检查避免了向已断开的连接发送消息（会导致 write error）。
func (h *Hub) SendToUser(userID int64, role string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client.userID == userID && client.role == role && client.IsActive() {
			select {
			case client.send <- message:
			default:
			}
		}
	}
}
