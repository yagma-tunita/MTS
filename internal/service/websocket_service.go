package service

import (
	"encoding/json"
	"time"

	ws "backend/net/websocket"
)

type WebSocketService interface {
	PushOrderStatusUpdate(userID int64, role string, orderID int64, newStatus int8) error
}

type webSocketServiceImpl struct{}

func NewWebSocketService() WebSocketService {
	return &webSocketServiceImpl{}
}

func (s *webSocketServiceImpl) PushOrderStatusUpdate(userID int64, role string, orderID int64, newStatus int8) error {
	payload := map[string]interface{}{
		"type":      "order_status_update",
		"order_id":  orderID,
		"status":    newStatus,
		"timestamp": time.Now().Unix(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	ws.PushToUser(userID, role, data)
	return nil
}
