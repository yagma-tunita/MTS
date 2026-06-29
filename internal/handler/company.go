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

type ShipperCompanyHandler struct {
	svc service.ShipperCompanyService
}

func NewShipperCompanyHandler(svc service.ShipperCompanyService) *ShipperCompanyHandler {
	return &ShipperCompanyHandler{svc: svc}
}

type registerShipperRequest struct {
	CompanyName   string `json:"company_name" validate:"required"`
	LoginUsername string `json:"login_username" validate:"required"`
	Password      string `json:"password" validate:"required,min=6"`
}

// RegisterShipper registers a new shipper company.
// @Summary      Register shipper company
// @Description  Create a new shipper account (public registration)
// @Tags         Shipper
// @Accept       json
// @Produce      json
// @Param        request body registerShipperRequest true "Shipper registration data"
// @Success      200 {object} response.Response{data=model.ShipperCompany} "Registration successful"
// @Failure      400 {object} response.Response "Bad request"
// @Router       /shipper/register [post]
func (h *ShipperCompanyHandler) Register(c *gin.Context) {
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
		CompanyName:   req.CompanyName,
		LoginUsername: req.LoginUsername,
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

type updatePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// UpdateShipperPassword changes the password of a shipper company.
// @Summary      Update shipper password
// @Description  Change password for an authenticated shipper
// @Tags         Shipper
// @Accept       json
// @Produce      json
// @Param        id path int true "Company ID"
// @Param        request body updatePasswordRequest true "Password update data"
// @Success      200 {object} response.Response{data=object{message=string}} "Password updated"
// @Failure      400 {object} response.Response "Bad request"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "Company not found"
// @Security     BearerAuth
// @Router       /shipper/password/{id} [post]
func (h *ShipperCompanyHandler) UpdatePassword(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid company id")
		return
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

type registerShippingRequest struct {
	CompanyName   string `json:"company_name" validate:"required"`
	LoginUsername string `json:"login_username" validate:"required"`
	Password      string `json:"password" validate:"required,min=6"`
}

type ShippingCompanyHandler struct {
	svc service.ShippingCompanyService
}

func NewShippingCompanyHandler(svc service.ShippingCompanyService) *ShippingCompanyHandler {
	return &ShippingCompanyHandler{svc: svc}
}

// RegisterShipping registers a new shipping company.
// @Summary      Register shipping company
// @Description  Create a new shipping company account (public registration)
// @Tags         Shipping
// @Accept       json
// @Produce      json
// @Param        request body registerShippingRequest true "Shipping company registration data"
// @Success      200 {object} response.Response{data=model.ShippingCompany} "Registration successful"
// @Failure      400 {object} response.Response "Bad request"
// @Router       /shipping/register [post]
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
		CompanyName:   req.CompanyName,
		LoginUsername: req.LoginUsername,
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

// UpdateShippingPassword changes the password of a shipping company.
// @Summary      Update shipping password
// @Description  Change password for an authenticated shipping company
// @Tags         Shipping
// @Accept       json
// @Produce      json
// @Param        id path int true "Company ID"
// @Param        request body updatePasswordRequest true "Password update data"
// @Success      200 {object} response.Response{data=object{message=string}} "Password updated"
// @Failure      400 {object} response.Response "Bad request"
// @Failure      401 {object} response.Response "Unauthorized"
// @Failure      404 {object} response.Response "Company not found"
// @Security     BearerAuth
// @Router       /shipping/password/{id} [post]
func (h *ShippingCompanyHandler) UpdatePassword(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid company id")
		return
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
