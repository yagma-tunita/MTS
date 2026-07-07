package handler

import (
	"strconv"

	"backend/internal/service"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// NotificationHandler 处理通知相关请求。
//
// 通知存储：当前使用内存 map（进程级），重启后丢失。
// 如果多实例部署，需改为 Redis 或数据库存储。
type NotificationHandler struct {
	svc service.NotificationService
}

// NewNotificationHandler 创建通知处理器
func NewNotificationHandler(svc service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// sendNotificationRequest 发送通知的请求体。
type sendNotificationRequest struct {
	UserID  int64                  `json:"user_id" validate:"required"`
	Role    string                 `json:"role" validate:"required,oneof=shipper shipping admin"`
	Type    string                 `json:"type" validate:"required"`
	Title   string                 `json:"title" validate:"required"`
	Content string                 `json:"content" validate:"required"`
	Data    map[string]interface{} `json:"data"` // 可选，含 email 字段发邮件，含 phone 字段发短信
}

// SendNotification 发送通知。
// 仅 admin 角色可调用（路由层 RequireRole("admin") 保护）。
// 通知同时会尝试异步发送邮件/SMS（如果 data 中提供了 email/phone）。
func (h *NotificationHandler) SendNotification(c *gin.Context) {
	var req sendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request")
		return
	}
	notif := &service.Notification{
		Type:     service.NotificationType(req.Type),
		UserID:   req.UserID,
		UserRole: req.Role,
		Title:    req.Title,
		Content:  req.Content,
		Data:     req.Data,
	}
	if err := h.svc.Send(c.Request.Context(), notif); err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	response.Success(c.Writer, gin.H{"message": "notification sent"})
}

// ListNotifications 获取当前用户的通知列表（分页）。
//
// 用户身份从 JWT Context 中获取（user_id + role），
// 而不是从请求参数中获取，防止越权查看他人的通知。
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c.Writer, "missing user context")
		return
	}
	userID := userIDRaw.(int64)
	roleRaw, _ := c.Get("role")
	role := roleRaw.(string)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	offset := (page - 1) * pageSize

	list, total, err := h.svc.GetUserNotifications(c.Request.Context(), userID, role, pageSize, offset)
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	response.SuccessPage(c.Writer, list, page, pageSize, total)
}

// MarkAsRead 将指定通知标记为已读。
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.MarkAsRead(c.Request.Context(), id); err != nil {
		response.NotFound(c.Writer, err.Error())
		return
	}
	response.Success(c.Writer, gin.H{"message": "marked as read"})
}
