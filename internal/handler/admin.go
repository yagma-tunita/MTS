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

type AdminHandler struct {
	svc service.AdminService
}

func NewAdminHandler(svc service.AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

type createAdminRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
	RealName string `json:"real_name"`
	Role     int8   `json:"role" validate:"omitempty,min=1,max=2"`
}

// CreateAdmin creates a new admin user (requires admin role).
// @Summary      Create admin
// @Description  Create a new administrator account (only accessible by existing admin)
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        request body createAdminRequest true "Admin creation data"
// @Success      200 {object} response.Response{data=model.Admin} "Admin created"
// @Failure      400 {object} response.Response "Bad request"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      403 {object} response.Response "Forbidden (requires admin role)"
// @Security     BearerAuth
// @Router       /admin/register [post]
func (h *AdminHandler) Create(c *gin.Context) {
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

type updateAdminPasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// UpdateAdminPassword changes password of an admin.
// @Summary      Update admin password
// @Description  Change password for an admin account
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        id path int true "Admin ID"
// @Param        request body updateAdminPasswordRequest true "Password update data"
// @Success      200 {object} response.Response{data=object{message=string}} "Password updated"
// @Failure      400 {object} response.Response "Bad request"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "Admin not found"
// @Security     BearerAuth
// @Router       /admin/password/{id} [post]
func (h *AdminHandler) UpdatePassword(c *gin.Context) {
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
