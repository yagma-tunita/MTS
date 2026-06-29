package handler

import (
	"backend/internal/service"
	"backend/pkg/errors"
	"backend/pkg/jwt"
	"backend/pkg/response"
	"backend/pkg/validator"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	shipperSvc  service.ShipperCompanyService
	shippingSvc service.ShippingCompanyService
	adminSvc    service.AdminService
	jwtSvc      jwt.JWTService
}

func NewAuthHandler(
	shipperSvc service.ShipperCompanyService,
	shippingSvc service.ShippingCompanyService,
	adminSvc service.AdminService,
	jwtSvc jwt.JWTService,
) *AuthHandler {
	return &AuthHandler{
		shipperSvc:  shipperSvc,
		shippingSvc: shippingSvc,
		adminSvc:    adminSvc,
		jwtSvc:      jwtSvc,
	}
}

type loginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	Role     string `json:"role" validate:"required,oneof=shipper shipping admin"`
}

// Login handles user login.
// @Summary      User login
// @Description  Authenticates user based on role and returns JWT tokens
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body loginRequest true "Login credentials"
// @Success      200 {object} response.Response{data=object{access_token=string,refresh_token=string,role=string,user_id=int}} "Login successful"
// @Failure      400 {object} response.Response "Bad request"
// @Failure      401 {object} response.Response "Unauthorized"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := validator.Validate(req); err != nil {
		response.BadRequest(c.Writer, err.Error())
		return
	}

	var userID int64
	var username string
	var role string

	switch req.Role {
	case "shipper":
		company, err := h.shipperSvc.Login(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			response.ErrorWithCode(c.Writer, errors.CodeUnauthorized, "invalid credentials")
			return
		}
		userID = company.CompanyID
		username = company.LoginUsername
		role = "shipper"
	case "shipping":
		company, err := h.shippingSvc.Login(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			response.ErrorWithCode(c.Writer, errors.CodeUnauthorized, "invalid credentials")
			return
		}
		userID = company.CompanyID
		username = company.LoginUsername
		role = "shipping"
	case "admin":
		admin, err := h.adminSvc.Login(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			response.ErrorWithCode(c.Writer, errors.CodeUnauthorized, "invalid credentials")
			return
		}
		userID = admin.AdminID
		username = admin.Username
		role = "admin"
	}

	accessToken, err := h.jwtSvc.GenerateAccessToken(userID, username, role)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to generate token")
		return
	}
	refreshToken, err := h.jwtSvc.GenerateRefreshToken(userID, username, role)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to generate refresh token")
		return
	}

	response.Success(c.Writer, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"role":          role,
		"user_id":       userID,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshToken returns a new access token using a valid refresh token.
// @Summary      Refresh access token
// @Description  Obtain a new access token using the refresh token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body refreshRequest true "Refresh token"
// @Success      200 {object} response.Response{data=object{access_token=string}} "Token refreshed"
// @Failure      400 {object} response.Response "Bad request"
// @Failure      401 {object} response.Response "Invalid refresh token"
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := validator.Validate(req); err != nil {
		response.BadRequest(c.Writer, err.Error())
		return
	}

	newAccessToken, err := h.jwtSvc.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		response.ErrorWithCode(c.Writer, errors.CodeUnauthorized, "invalid refresh token")
		return
	}

	response.Success(c.Writer, gin.H{"access_token": newAccessToken})
}
