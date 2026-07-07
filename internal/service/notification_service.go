package service

import (
	"context"
	"fmt"
	"time"

	"backend/internal/notify"
)

// NotificationType 通知类型枚举。
type NotificationType string

const (
	NotificationOrderCreated   NotificationType = "order_created"
	NotificationOrderCancelled NotificationType = "order_cancelled"
	NotificationStatusChanged  NotificationType = "status_changed"
)

// Notification 通知数据结构。
type Notification struct {
	ID        string                 `json:"id"`
	Type      NotificationType       `json:"type"`
	UserID    int64                  `json:"user_id"`
	UserRole  string                 `json:"user_role"`
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	Data      map[string]interface{} `json:"data,omitempty"`
	CreatedAt time.Time              `json:"create_time"`
	Read      bool                   `json:"read"`
}

// NotificationService 通知服务接口。
type NotificationService interface {
	Send(ctx context.Context, notif *Notification) error
	GetUserNotifications(ctx context.Context, userID int64, role string, limit, offset int) ([]Notification, int64, error)
	MarkAsRead(ctx context.Context, id string) error
}

// notificationServiceImpl 使用内存存储的通知服务实现。
// 注意：通知存储在进程内存中，重启后丢失。多实例部署应替换为 Redis 或数据库。
type notificationServiceImpl struct {
	storage map[string][]Notification
	prov    *notify.Provider
}

// NewNotificationService 创建通知服务实例
func NewNotificationService(prov *notify.Provider) NotificationService {
	return &notificationServiceImpl{
		storage: make(map[string][]Notification),
		prov:    prov,
	}
}

// Send 存储通知到内存，并异步发送邮件/SMS（如果配置了通知提供商）。
func (s *notificationServiceImpl) Send(ctx context.Context, notif *Notification) error {
	if notif.ID == "" {
		notif.ID = fmt.Sprintf("notif_%d", time.Now().UnixNano())
	}
	notif.CreatedAt = time.Now()
	key := fmt.Sprintf("%d:%s", notif.UserID, notif.UserRole)
	s.storage[key] = append(s.storage[key], *notif)
	Logger.Info("notification sent", "user_id", notif.UserID, "type", notif.Type, "title", notif.Title)

	// 异步发送邮件 / SMS
	if s.prov != nil && notif.Data != nil {
		if email, ok := notif.Data["email"].(string); ok && email != "" && s.prov.Email != nil && s.prov.Email.IsConfigured() {
			go func() {
				if err := s.prov.Email.Send(email, notif.Title, notif.Content); err != nil {
					Logger.Error("email send failed", "to", email, "error", err)
				}
			}()
		}
		if phone, ok := notif.Data["phone"].(string); ok && phone != "" && s.prov.SMS != nil {
			go func() {
				if err := s.prov.SMS.Send(phone, notif.Content); err != nil {
					Logger.Error("sms send failed", "to", phone, "error", err)
				}
			}()
		}
	}
	return nil
}

// GetUserNotifications 获取指定用户的通知列表（倒序，最新的在前），支持分页。
func (s *notificationServiceImpl) GetUserNotifications(ctx context.Context, userID int64, role string, limit, offset int) ([]Notification, int64, error) {
	key := fmt.Sprintf("%d:%s", userID, role)
	list := s.storage[key]
	total := int64(len(list))
	start := offset
	end := offset + limit
	if start > len(list) {
		start = len(list)
	}
	if end > len(list) {
		end = len(list)
	}
	result := make([]Notification, 0, end-start)
	for i := end - 1; i >= start; i-- {
		result = append(result, list[i])
	}
	return result, total, nil
}

// MarkAsRead 将指定通知标记为已读。
func (s *notificationServiceImpl) MarkAsRead(ctx context.Context, id string) error {
	for key, list := range s.storage {
		for i := range list {
			if list[i].ID == id {
				s.storage[key][i].Read = true
				return nil
			}
		}
	}
	return fmt.Errorf("notification not found")
}

