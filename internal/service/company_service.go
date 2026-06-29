package service

import (
	"context"

	"backend/internal/dao"
	"backend/internal/model"
	"backend/pkg/crypto"
	pkgerr "backend/pkg/errors"
)

type ShipperCompanyService interface {
	Register(ctx context.Context, company *model.ShipperCompany, plainPassword string) error
	Login(ctx context.Context, username, plainPassword string) (*model.ShipperCompany, error)
	UpdatePassword(ctx context.Context, companyID int64, oldPassword, newPassword string) error
}

type shipperCompanyServiceImpl struct {
	dao dao.ShipperCompanyDAO
}

func NewShipperCompanyService(dao dao.ShipperCompanyDAO) ShipperCompanyService {
	return &shipperCompanyServiceImpl{dao: dao}
}

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

func (s *shipperCompanyServiceImpl) Login(ctx context.Context, username, plainPassword string) (*model.ShipperCompany, error) {
	logger := Logger.With("method", "ShipperCompanyLogin", "username", username)
	logger.Debug("shipper company login")

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
	logger.Info("shipper company logged in", "company_id", company.CompanyID)
	return company, nil
}

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

type ShippingCompanyService interface {
	Register(ctx context.Context, company *model.ShippingCompany, plainPassword string) error
	Login(ctx context.Context, username, plainPassword string) (*model.ShippingCompany, error)
	UpdatePassword(ctx context.Context, companyID int64, oldPassword, newPassword string) error
}

type shippingCompanyServiceImpl struct {
	dao dao.ShippingCompanyDAO
}

func NewShippingCompanyService(dao dao.ShippingCompanyDAO) ShippingCompanyService {
	return &shippingCompanyServiceImpl{dao: dao}
}

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
