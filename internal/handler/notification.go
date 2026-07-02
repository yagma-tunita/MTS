package handler

import (
	"strconv"

	"backend/internal/service"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	svc service.NotificationService
}

func NewNotificationHandler(svc service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

type sendNotificationRequest struct {
	UserID  int64                  `json:"user_id" validate:"required"`
	Role    string                 `json:"role" validate:"required,oneof=shipper shipping admin"`
	Type    string                 `json:"type" validate:"required"`
	Title   string                 `json:"title" validate:"required"`
	Content string                 `json:"content" validate:"required"`
	Data    map[string]interface{} `json:"data"`
}

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

func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.MarkAsRead(c.Request.Context(), id); err != nil {
		response.NotFound(c.Writer, err.Error())
		return
	}
	response.Success(c.Writer, gin.H{"message": "marked as read"})
}
