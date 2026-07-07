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

// ShipperCompanyHandler 处理货主公司相关请求。
type ShipperCompanyHandler struct {
	svc service.ShipperCompanyService
}

// NewShipperCompanyHandler 创建货主公司处理器
func NewShipperCompanyHandler(svc service.ShipperCompanyService) *ShipperCompanyHandler {
	return &ShipperCompanyHandler{svc: svc}
}

// shipperListRequest 货主公司列表查询参数。
type shipperListRequest struct {
	Page     int `form:"page" validate:"omitempty,min=1"`
	PageSize int `form:"page_size" validate:"omitempty,min=1,max=100"`
}

// List 管理员分页查询货主公司列表。
func (h *ShipperCompanyHandler) List(c *gin.Context) {
	var req shipperListRequest
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
	companies, total, err := h.svc.List(c.Request.Context(), req.Page, req.PageSize)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list shipper companies")
		return
	}
	response.SuccessPage(c.Writer, companies, req.Page, req.PageSize, total)
}

// UpdateByAdmin 管理员更新货主公司信息（仅更新请求中传了的字段）。
func (h *ShipperCompanyHandler) UpdateByAdmin(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid company id")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := h.svc.UpdateByAdmin(c.Request.Context(), id, req); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to update shipper company")
		return
	}
	response.Success(c.Writer, gin.H{"message": "shipper company updated"})
}

// registerShipperRequest 货主注册请求体。
type registerShipperRequest struct {
	CompanyName             string  `json:"company_name" validate:"required"`
	LoginUsername           string  `json:"login_username" validate:"required"`
	Password                string  `json:"password" validate:"required,min=6"`
	UnifiedSocialCreditCode *string `json:"unified_social_credit_code"`
	LegalRepresentative     *string `json:"legal_representative"`
	ContactPhone            *string `json:"contact_phone"`
	Address                 *string `json:"address"`
}

func (h *ShipperCompanyHandler) Register(c *gin.Context) { // 注册货主公司账号
	var req registerShipperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := validator.Validate(req); err != nil {
		response.BadRequest(c.Writer, err.Error())
		return
	}

	company := &model.ShipperCompany{
		CompanyName:             req.CompanyName,
		LoginUsername:           req.LoginUsername,
		UnifiedSocialCreditCode: req.UnifiedSocialCreditCode,
		LegalRepresentative:     req.LegalRepresentative,
		ContactPhone:            req.ContactPhone,
		Address:                 req.Address,
	}
	if err := h.svc.Register(c.Request.Context(), company, req.Password); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to register")
		return
	}
	response.Success(c.Writer, company)
}

// updatePasswordRequest 改密请求体（货主和船公司共用）。
type updatePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// UpdatePassword 修改货主公司密码。
//
// {id} 必须等于当前 JWT 中的 user_id（shipper 角色时强制校验）。
// 需要提供旧密码进行验证。
func (h *ShipperCompanyHandler) UpdatePassword(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "shipper" && role != "admin" {
		response.ErrorWithCode(c.Writer, errors.CodeForbidden, "only shipper can change password"); return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid company id")
		return
	}

	userID, _ := c.Get("user_id")
	if role == "shipper" {
		if uid, ok := userID.(int64); !ok || uid != id {
			response.ErrorWithCode(c.Writer, errors.CodeForbidden, "can only update your own password")
			return
		}
	}

	var req updatePasswordRequest
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

// registerShippingRequest 船公司注册请求体。
type registerShippingRequest struct {
	CompanyName             string  `json:"company_name" validate:"required"`
	LoginUsername           string  `json:"login_username" validate:"required"`
	Password                string  `json:"password" validate:"required,min=6"`
	UnifiedSocialCreditCode *string `json:"unified_social_credit_code"`
	ContactPerson           *string `json:"contact_person"`
	ContactPhone            *string `json:"contact_phone"`
	Address                 *string `json:"address"`
}

// ShippingCompanyHandler 处理船公司相关请求。
type ShippingCompanyHandler struct {
	svc service.ShippingCompanyService
}

// NewShippingCompanyHandler 创建船公司处理器
func NewShippingCompanyHandler(svc service.ShippingCompanyService) *ShippingCompanyHandler {
	return &ShippingCompanyHandler{svc: svc}
}

// shippingListRequest 船公司列表查询参数。
type shippingListRequest struct {
	Page     int `form:"page" validate:"omitempty,min=1"`
	PageSize int `form:"page_size" validate:"omitempty,min=1,max=100"`
}

// List 管理员分页查询船公司列表。
func (h *ShippingCompanyHandler) List(c *gin.Context) {
	var req shippingListRequest
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
	companies, total, err := h.svc.List(c.Request.Context(), req.Page, req.PageSize)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list shipping companies")
		return
	}
	response.SuccessPage(c.Writer, companies, req.Page, req.PageSize, total)
}

// UpdateByAdmin 管理员更新船公司信息（仅更新请求中传了的字段）。
func (h *ShippingCompanyHandler) UpdateByAdmin(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid company id")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := h.svc.UpdateByAdmin(c.Request.Context(), id, req); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to update shipping company")
		return
	}
	response.Success(c.Writer, gin.H{"message": "shipping company updated"})
}

// Register 注册船公司账号（公开接口）。
func (h *ShippingCompanyHandler) Register(c *gin.Context) {
	var req registerShippingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := validator.Validate(req); err != nil {
		response.BadRequest(c.Writer, err.Error())
		return
	}
	company := &model.ShippingCompany{
		CompanyName:             req.CompanyName,
		LoginUsername:           req.LoginUsername,
		UnifiedSocialCreditCode: req.UnifiedSocialCreditCode,
		ContactPerson:           req.ContactPerson,
		ContactPhone:            req.ContactPhone,
		Address:                 req.Address,
	}
	if err := h.svc.Register(c.Request.Context(), company, req.Password); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to register")
		return
	}
	response.Success(c.Writer, company)
}

// UpdatePassword 修改船公司密码。
//
// {id} 必须等于当前 JWT 中的 user_id（shipping 角色时强制校验）。
// 需要提供旧密码进行验证。
func (h *ShippingCompanyHandler) UpdatePassword(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "shipping" && role != "admin" {
		response.ErrorWithCode(c.Writer, errors.CodeForbidden, "only shipping can change password"); return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid company id")
		return
	}

	userID, _ := c.Get("user_id")
	if role == "shipping" {
		if uid, ok := userID.(int64); !ok || uid != id {
			response.ErrorWithCode(c.Writer, errors.CodeForbidden, "can only update your own password")
			return
		}
	}

	var req updatePasswordRequest
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

// Delete 软删除货主公司（admin 权限）。
//
// 设置 delete_time 后该账号无法登录。
// 仅限 admin 角色通过路由 /admin/shipper/{id}/delete 访问。
func (h *ShipperCompanyHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid company id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to delete shipper company")
		return
	}
	response.Success(c.Writer, gin.H{"message": "shipper company deleted"})
}

// Delete 软删除船公司（admin 权限）。
func (h *ShippingCompanyHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid company id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			response.ErrorWithCode(c.Writer, appErr.Code, appErr.Message)
			return
		}
		response.InternalServerError(c.Writer, "failed to delete shipping company")
		return
	}
	response.Success(c.Writer, gin.H{"message": "shipping company deleted"})
}
