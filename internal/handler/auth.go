package handler

import (
	"backend/internal/service"
	"backend/pkg/errors"
	"backend/pkg/jwt"
	"backend/pkg/response"
	"backend/pkg/validator"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct { // 用户认证处理器（三角色共用登录接口）
	shipperSvc  service.ShipperCompanyService
	shippingSvc service.ShippingCompanyService
	adminSvc    service.AdminService
	jwtSvc      jwt.JWTService
}

func NewAuthHandler( // 创建认证处理器
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

type loginRequest struct { // 登录请求体（三角色共用）
	Username string `json:"username" validate:"required"`                 // 用户名
	Password string `json:"password" validate:"required"`                 // 密码
	Role     string `json:"role" validate:"required,oneof=shipper shipping admin"` // 角色
}

func (h *AuthHandler) Login(c *gin.Context) { // 处理用户登录请求（三角色分流）
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
	var companyName string

	switch req.Role {
	case "shipper":
		company, err := h.shipperSvc.Login(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			response.ErrorWithCode(c.Writer, errors.CodeUnauthorized, "invalid credentials")
			return
		}
		userID = company.CompanyID
		username = company.LoginUsername
		companyName = company.CompanyName
		role = "shipper"
	case "shipping":
		company, err := h.shippingSvc.Login(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			response.ErrorWithCode(c.Writer, errors.CodeUnauthorized, "invalid credentials")
			return
		}
		userID = company.CompanyID
		username = company.LoginUsername
		companyName = company.CompanyName
		role = "shipping"
	case "admin":
		admin, err := h.adminSvc.Login(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			response.ErrorWithCode(c.Writer, errors.CodeUnauthorized, "invalid credentials")
			return
		}
		userID = admin.AdminID
		username = admin.Username
		if admin.RealName != nil {
			companyName = *admin.RealName
		}
		if companyName == "" {
			companyName = admin.Username
		}
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
		"company_name":  companyName,
	})
}

type refreshRequest struct { // 刷新令牌的请求体
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (h *AuthHandler) Me(c *gin.Context) { // 返回当前登录用户信息
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	// Note: company_name 与 username 的区别是设计上的——登录接口返回 company_name
	// 用于展示企业名称，而 /auth/me 返回的 username 是登录账号名，二者有意不同。
	response.Success(c.Writer, gin.H{
		"user_id":  userID,
		"username": username,
		"role":     role,
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) { // 刷新 access_token
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

