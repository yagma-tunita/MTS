package service

import (
	"fmt"

	"backend/internal/dao"
	"backend/internal/model"
	"backend/pkg/crypto"
	pkgerr "backend/pkg/errors"
	"context"
	"errors"

	"gorm.io/gorm"
)

// ShipperCompanyService 货主公司服务接口。
type ShipperCompanyService interface {
	Register(ctx context.Context, company *model.ShipperCompany, plainPassword string) error
	Login(ctx context.Context, username, plainPassword string) (*model.ShipperCompany, error)
	UpdatePassword(ctx context.Context, companyID int64, oldPassword, newPassword string) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, page, pageSize int) ([]model.ShipperCompany, int64, error)
	GetByID(ctx context.Context, id int64) (*model.ShipperCompany, error)
	UpdateByAdmin(ctx context.Context, id int64, updates map[string]interface{}) error
}

// shipperCompanyServiceImpl 是 ShipperCompanyService 接口的私有实现。
type shipperCompanyServiceImpl struct {
	dao dao.ShipperCompanyDAO
}

// NewShipperCompanyService 创建货主公司服务实例
func NewShipperCompanyService(dao dao.ShipperCompanyDAO) ShipperCompanyService {
	return &shipperCompanyServiceImpl{dao: dao}
}

// Register 注册新货主公司，密码使用 bcrypt 哈希后存储。
func (s *shipperCompanyServiceImpl) Register(ctx context.Context, company *model.ShipperCompany, plainPassword string) error {
	logger := Logger.With("method", "RegisterShipperCompany", "username", company.LoginUsername)
	logger.Debug("registering shipper company")

	hash, err := crypto.HashPassword(plainPassword)
	if err != nil {
		logger.Error("failed to hash password", "error", err)
		return err
	}
	company.LoginPassword = hash
	if err := s.dao.Create(company); err != nil {
		logger.Error("failed to create shipper company", "error", err)
		return err
	}
	logger.Info("shipper company registered", "company_id", company.CompanyID)
	return nil
}

// Login 验证货主公司用户名密码，检查账号状态后返回公司信息。
func (s *shipperCompanyServiceImpl) Login(ctx context.Context, username, plainPassword string) (*model.ShipperCompany, error) {
	fmt.Println("=== Login method called for user:", username)
	logger := Logger.With("method", "ShipperCompanyLogin", "username", username)
	logger.Debug("shipper company login")

	company, err := s.dao.GetByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("company not found")
		} else {
			logger.Error("GetByUsername failed", "error", err)
		}
		return nil, pkgerr.Unauthorized("invalid username or password")
	}
	if !crypto.CheckPasswordHash(plainPassword, company.LoginPassword) {
		logger.Warn("invalid password")
		return nil, pkgerr.Unauthorized("invalid username or password")
	}
	if company.AccountStatus != 1 {
		logger.Warn("account disabled", "status", company.AccountStatus)
		return nil, pkgerr.Forbidden("account disabled")
	}
	logger.Info("shipper company logged in", "company_id", company.CompanyID)
	return company, nil
}

// UpdatePassword 验证旧密码后更新货主公司密码。
func (s *shipperCompanyServiceImpl) UpdatePassword(ctx context.Context, companyID int64, oldPassword, newPassword string) error {
	logger := Logger.With("method", "UpdateShipperCompanyPassword", "company_id", companyID)
	logger.Debug("updating shipper company password")

	company, err := s.dao.GetByID(companyID)
	if err != nil {
		logger.Error("company not found", "error", err)
		return pkgerr.NotFound("company not found")
	}
	if !crypto.CheckPasswordHash(oldPassword, company.LoginPassword) {
		logger.Warn("wrong old password")
		return pkgerr.BadRequest("wrong old password")
	}
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		logger.Error("failed to hash new password", "error", err)
		return err
	}
	company.LoginPassword = hash
	if err := s.dao.Update(company); err != nil {
		logger.Error("failed to update company", "error", err)
		return err
	}
	logger.Info("shipper company password updated")
	return nil
}

// List 分页查询货主公司列表（admin 用）。
func (s *shipperCompanyServiceImpl) List(ctx context.Context, page, pageSize int) ([]model.ShipperCompany, int64, error) {
	return s.dao.List(page, pageSize)
}

// GetByID 查询货主公司。
func (s *shipperCompanyServiceImpl) GetByID(ctx context.Context, id int64) (*model.ShipperCompany, error) {
	return s.dao.GetByID(id)
}

// UpdateByAdmin 管理员更新货主公司信息（仅更新请求中传了的字段）。
func (s *shipperCompanyServiceImpl) UpdateByAdmin(ctx context.Context, id int64, updates map[string]interface{}) error {
	if _, err := s.dao.GetByID(id); err != nil {
		return pkgerr.NotFound("shipper company not found")
	}
	return s.dao.UpdateMap(id, updates)
}

// Delete 软删除货主公司（设置 delete_time）。
func (s *shipperCompanyServiceImpl) Delete(ctx context.Context, id int64) error {
	logger := Logger.With("method", "DeleteShipperCompany", "company_id", id)
	logger.Debug("deleting shipper company")

	if _, err := s.dao.GetByID(id); err != nil {
		logger.Warn("shipper company not found")
		return pkgerr.NotFound("shipper company not found")
	}
	if err := s.dao.Delete(id); err != nil {
		logger.Error("failed to delete shipper company", "error", err)
		return err
	}
	logger.Info("shipper company deleted")
	return nil
}

// ShippingCompanyService 船公司服务接口。
type ShippingCompanyService interface {
	Register(ctx context.Context, company *model.ShippingCompany, plainPassword string) error
	Login(ctx context.Context, username, plainPassword string) (*model.ShippingCompany, error)
	UpdatePassword(ctx context.Context, companyID int64, oldPassword, newPassword string) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, page, pageSize int) ([]model.ShippingCompany, int64, error)
	UpdateByAdmin(ctx context.Context, id int64, updates map[string]interface{}) error
}

// shippingCompanyServiceImpl 是 ShippingCompanyService 接口的私有实现。
type shippingCompanyServiceImpl struct {
	dao dao.ShippingCompanyDAO
}

// NewShippingCompanyService 创建船公司服务实例
func NewShippingCompanyService(dao dao.ShippingCompanyDAO) ShippingCompanyService {
	return &shippingCompanyServiceImpl{dao: dao}
}

// Register 注册新船公司，密码使用 bcrypt 哈希后存储。
func (s *shippingCompanyServiceImpl) Register(ctx context.Context, company *model.ShippingCompany, plainPassword string) error {
	logger := Logger.With("method", "RegisterShippingCompany", "username", company.LoginUsername)
	logger.Debug("registering shipping company")

	hash, err := crypto.HashPassword(plainPassword)
	if err != nil {
		logger.Error("failed to hash password", "error", err)
		return err
	}
	company.LoginPassword = hash
	if err := s.dao.Create(company); err != nil {
		logger.Error("failed to create shipping company", "error", err)
		return err
	}
	logger.Info("shipping company registered", "company_id", company.CompanyID)
	return nil
}

// Login 验证船公司用户名密码，检查账号状态后返回公司信息。
func (s *shippingCompanyServiceImpl) Login(ctx context.Context, username, plainPassword string) (*model.ShippingCompany, error) {
	logger := Logger.With("method", "ShippingCompanyLogin", "username", username)
	logger.Debug("shipping company login")

	company, err := s.dao.GetByUsername(username)
	if err != nil {
		logger.Warn("company not found")
		return nil, pkgerr.Unauthorized("invalid username or password")
	}
	if !crypto.CheckPasswordHash(plainPassword, company.LoginPassword) {
		logger.Warn("invalid password")
		return nil, pkgerr.Unauthorized("invalid username or password")
	}
	if company.AccountStatus != 1 {
		logger.Warn("account disabled", "status", company.AccountStatus)
		return nil, pkgerr.Forbidden("account disabled")
	}
	logger.Info("shipping company logged in", "company_id", company.CompanyID)
	return company, nil
}

// UpdatePassword 验证旧密码后更新船公司密码。
func (s *shippingCompanyServiceImpl) UpdatePassword(ctx context.Context, companyID int64, oldPassword, newPassword string) error {
	logger := Logger.With("method", "UpdateShippingCompanyPassword", "company_id", companyID)
	logger.Debug("updating shipping company password")

	company, err := s.dao.GetByID(companyID)
	if err != nil {
		logger.Error("company not found", "error", err)
		return pkgerr.NotFound("company not found")
	}
	if !crypto.CheckPasswordHash(oldPassword, company.LoginPassword) {
		logger.Warn("wrong old password")
		return pkgerr.BadRequest("wrong old password")
	}
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		logger.Error("failed to hash new password", "error", err)
		return err
	}
	company.LoginPassword = hash
	if err := s.dao.Update(company); err != nil {
		logger.Error("failed to update company", "error", err)
		return err
	}
	logger.Info("shipping company password updated")
	return nil
}

// List 分页查询船公司列表（admin 用）。
func (s *shippingCompanyServiceImpl) List(ctx context.Context, page, pageSize int) ([]model.ShippingCompany, int64, error) {
	return s.dao.List(page, pageSize)
}

// UpdateByAdmin 管理员更新船公司信息。
func (s *shippingCompanyServiceImpl) UpdateByAdmin(ctx context.Context, id int64, updates map[string]interface{}) error {
	if _, err := s.dao.GetByID(id); err != nil {
		return pkgerr.NotFound("shipping company not found")
	}
	return s.dao.UpdateMap(id, updates)
}

// Delete 软删除船公司（设置 delete_time）。
func (s *shippingCompanyServiceImpl) Delete(ctx context.Context, id int64) error {
	logger := Logger.With("method", "DeleteShippingCompany", "company_id", id)
	logger.Debug("deleting shipping company")

	if _, err := s.dao.GetByID(id); err != nil {
		logger.Warn("shipping company not found")
		return pkgerr.NotFound("shipping company not found")
	}
	if err := s.dao.Delete(id); err != nil {
		logger.Error("failed to delete shipping company", "error", err)
		return err
	}
	logger.Info("shipping company deleted")
	return nil
}

