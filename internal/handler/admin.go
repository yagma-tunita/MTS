package handler

import (
	"strconv"

	"backend/internal/model"
	"backend/internal/service"
	"backend/pkg/errors"
	"backend/pkg/response"
	"backend/pkg/validator"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct { // 管理员管理处理器（仅限 admin 角色访问）
	svc service.AdminService
}

func NewAdminHandler(svc service.AdminService) *AdminHandler { // 创建管理员处理器
	return &AdminHandler{svc: svc}
}

type adminListRequest struct { // 管理员列表查询参数
	Page     int `form:"page" validate:"omitempty,min=1"`
	PageSize int `form:"page_size" validate:"omitempty,min=1,max=100"`
}

func (h *AdminHandler) List(c *gin.Context) { // 管理员分页查询管理员列表
	var req adminListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c.Writer, "invalid query parameters")
		return
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}
	admins, total, err := h.svc.List(c.Request.Context(), req.Page, req.PageSize)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list admins")
		return
	}
	response.SuccessPage(c.Writer, admins, req.Page, req.PageSize, total)
}

type createAdminRequest struct { // 创建管理员的请求体
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
	RealName string `json:"real_name"`
	Role     int8   `json:"role" validate:"omitempty,min=1,max=2"`
}

func (h *AdminHandler) Create(c *gin.Context) { // 创建新的管理员账号
	var req createAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := validator.Validate(req); err != nil {
		response.BadRequest(c.Writer, err.Error())
		return
	}
	admin := &model.Admin{
		Username: req.Username,
		RealName: &req.RealName,
		Role:     req.Role,
	}
	if err := h.svc.Create(c.Request.Context(), admin, req.Password); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to create admin")
		return
	}
	response.Success(c.Writer, admin)
}

type updateAdminPasswordRequest struct { // 修改管理员密码的请求体
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

func (h *AdminHandler) UpdatePassword(c *gin.Context) { // 修改指定管理员的密码
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid admin id")
		return
	}
	var req updateAdminPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := validator.Validate(req); err != nil {
		response.BadRequest(c.Writer, err.Error())
		return
	}
	err = h.svc.UpdatePassword(c.Request.Context(), id, req.OldPassword, req.NewPassword)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to update password")
		return
	}
	response.Success(c.Writer, gin.H{"message": "password updated"})
}
