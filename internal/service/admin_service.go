package service

import (
	"context"

	"backend/internal/dao"
	"backend/internal/model"
	"backend/pkg/crypto"
	pkgerr "backend/pkg/errors"
)

// AdminService 管理员服务接口，包含创建、登录、修改密码、列表。
type AdminService interface {
	Create(ctx context.Context, admin *model.Admin, plainPassword string) error
	Login(ctx context.Context, username, plainPassword string) (*model.Admin, error)
	UpdatePassword(ctx context.Context, adminID int64, oldPassword, newPassword string) error
	List(ctx context.Context, page, pageSize int) ([]model.Admin, int64, error)
}

// adminServiceImpl 是 AdminService 接口的私有实现。
type adminServiceImpl struct {
	dao dao.AdminDAO
}

// NewAdminService 创建管理员服务
func NewAdminService(dao dao.AdminDAO) AdminService {
	return &adminServiceImpl{dao: dao}
}

// Create 创建管理员账号，使用 bcrypt 哈希密码后存储。
func (s *adminServiceImpl) Create(ctx context.Context, admin *model.Admin, plainPassword string) error {
	logger := Logger.With("method", "CreateAdmin", "username", admin.Username)
	logger.Debug("creating admin")

	hash, err := crypto.HashPassword(plainPassword)
	if err != nil {
		logger.Error("failed to hash password", "error", err)
		return err
	}
	plainPassword = ""
	admin.Password = hash
	if err := s.dao.Create(admin); err != nil {
		logger.Error("failed to create admin", "error", err)
		return err
	}
	logger.Info("admin created", "admin_id", admin.AdminID)
	return nil
}

// Login 验证管理员用户名密码，成功后返回 Admin 对象。
func (s *adminServiceImpl) Login(ctx context.Context, username, plainPassword string) (*model.Admin, error) {
	logger := Logger.With("method", "AdminLogin", "username", username)
	logger.Debug("admin login")

	admin, err := s.dao.GetByUsername(username)
	if err != nil {
		logger.Warn("admin not found")
		return nil, pkgerr.Unauthorized("invalid username or password")
	}
	if !crypto.CheckPasswordHash(plainPassword, admin.Password) {
		logger.Warn("invalid password")
		return nil, pkgerr.Unauthorized("invalid username or password")
	}
	logger.Info("admin logged in", "admin_id", admin.AdminID)
	return admin, nil
}

// List 分页查询管理员列表。
func (s *adminServiceImpl) List(ctx context.Context, page, pageSize int) ([]model.Admin, int64, error) {
	return s.dao.List(page, pageSize)
}

// UpdatePassword 验证旧密码后更新管理员密码。
func (s *adminServiceImpl) UpdatePassword(ctx context.Context, adminID int64, oldPassword, newPassword string) error {
	logger := Logger.With("method", "UpdateAdminPassword", "admin_id", adminID)
	logger.Debug("updating admin password")

	admin, err := s.dao.GetByID(adminID)
	if err != nil {
		logger.Error("admin not found", "error", err)
		return pkgerr.NotFound("admin not found")
	}
	if !crypto.CheckPasswordHash(oldPassword, admin.Password) {
		logger.Warn("wrong old password")
		return pkgerr.BadRequest("wrong old password")
	}
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		logger.Error("failed to hash new password", "error", err)
		return err
	}
	admin.Password = hash
	if err := s.dao.Update(admin); err != nil {
		logger.Error("failed to update admin", "error", err)
		return err
	}
	logger.Info("admin password updated")
	return nil
}

