package service

import (
	"encoding/json"
	"time"

	ws "backend/net/websocket"
)

// WebSocketService WebSocket 推送服务接口。
//
// 职责：向指定用户的 WebSocket 连接推送实时消息。
// 当前仅用于订单状态变更通知。如果将来需要推送其他类型
// 的消息（如系统公告、新订单提醒），可以在此接口中添加方法。
type WebSocketService interface {
	// PushOrderStatusUpdate 向指定用户推送订单状态变更通知。
	//
	// 参数：
	//   - userID: 目标用户的 ID（货主公司 ID / 船公司 ID / 管理员 ID）。
	//   - role: 目标用户的角色（shipper/shipping/admin）。
	//   - orderID: 发生状态变更的订单 ID。
	//   - newStatus: 变更后的状态值（0-4）。
	//
	// 内部调用 net/websocket.PushToUser，该函数遍历 Hub 中所有
	// 匹配 userID+role 的 WebSocket 连接并发送 JSON 消息。
	PushOrderStatusUpdate(userID int64, role string, orderID int64, newStatus int8) error
}

// webSocketServiceImpl 是 WebSocketService 接口的私有实现。
type webSocketServiceImpl struct{}

// NewWebSocketService 创建 WebSocket 推送服务
func NewWebSocketService() WebSocketService {
	return &webSocketServiceImpl{}
}

// PushOrderStatusUpdate 向指定用户推送订单状态变更通知。
//
// 推送的 JSON 消息格式：
//
//	{
//	  "type": "order_status_update",  // 消息类型，前端据此判断如何解析
//	  "order_id": 1,                  // 发生变更的订单 ID
//	  "status": 2,                    // 变更后的状态码
//	  "timestamp": 1741234567         // 推送时的 Unix 时间戳（秒）
//	}
//
// 前端可以在 WebSocket onmessage 事件中解析此消息并更新 UI。
// 建议前端根据 order_id 找到对应的订单列表项，更新其状态显示。
func (s *webSocketServiceImpl) PushOrderStatusUpdate(userID int64, role string, orderID int64, newStatus int8) error {
	payload := map[string]interface{}{
		"type":      "order_status_update",
		"order_id":  orderID,
		"status":    newStatus,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	ws.PushToUser(userID, role, data)
	return nil
}

