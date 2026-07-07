package service

import (
	"context"

	"backend/internal/dao"
	"backend/internal/model"
	pkgerr "backend/pkg/errors"
)

// VesselService 船舶服务接口，支持查询单个、列表、按公司查询、增删改。
type VesselService interface {
	GetVesselByID(ctx context.Context, id int64) (*model.Vessel, error)
	ListVessels(ctx context.Context, page, pageSize int, keyword string) ([]model.Vessel, int64, error)
	ListVesselsByCompany(ctx context.Context, companyID int64, page, pageSize int) ([]model.Vessel, int64, error)
	CreateVessel(ctx context.Context, vessel *model.Vessel) error
	UpdateVessel(ctx context.Context, vessel *model.Vessel) error
	DeleteVessel(ctx context.Context, id int64) error
}

// vesselServiceImpl 是 VesselService 接口的私有实现。
type vesselServiceImpl struct {
	dao dao.VesselDAO
}

// NewVesselService 创建船舶查询服务
func NewVesselService(dao dao.VesselDAO) VesselService {
	return &vesselServiceImpl{dao: dao}
}

// GetVesselByID 查询船舶详情。
func (s *vesselServiceImpl) GetVesselByID(ctx context.Context, id int64) (*model.Vessel, error) {
	logger := Logger.With("method", "GetVesselByID", "vessel_id", id)
	logger.Debug("fetching vessel")

	v, err := s.dao.GetByID(id)
	if err != nil {
		logger.Error("vessel not found", "error", err)
		return nil, pkgerr.NotFound("vessel not found")
	}
	return v, nil
}

// ListVessels 分页查询船舶列表。
func (s *vesselServiceImpl) ListVessels(ctx context.Context, page, pageSize int, keyword string) ([]model.Vessel, int64, error) {
	logger := Logger.With("method", "ListVessels", "page", page, "page_size", pageSize)
	logger.Debug("listing vessels")

	vessels, total, err := s.dao.List(page, pageSize, keyword)
	if err != nil {
		logger.Error("failed to list vessels", "error", err)
		return nil, 0, err
	}
	logger.Debug("vessels listed", "count", len(vessels), "total", total)
	return vessels, total, nil
}

// ListVesselsByCompany 根据船公司 ID 分页查询该公司船舶。
func (s *vesselServiceImpl) ListVesselsByCompany(ctx context.Context, companyID int64, page, pageSize int) ([]model.Vessel, int64, error) {
	logger := Logger.With("method", "ListVesselsByCompany", "company_id", companyID, "page", page, "page_size", pageSize)
	logger.Debug("listing vessels by company")

	vessels, total, err := s.dao.ListByShippingCompany(companyID, page, pageSize)
	if err != nil {
		logger.Error("failed to list vessels by company", "error", err)
		return nil, 0, err
	}
	logger.Debug("vessels listed", "count", len(vessels), "total", total)
	return vessels, total, nil
}

// CreateVessel 创建新船舶。
func (s *vesselServiceImpl) CreateVessel(ctx context.Context, vessel *model.Vessel) error {
	return s.dao.Create(vessel)
}

// UpdateVessel 更新船舶信息。
func (s *vesselServiceImpl) UpdateVessel(ctx context.Context, vessel *model.Vessel) error {
	return s.dao.Update(vessel)
}

// DeleteVessel 软删除船舶。
func (s *vesselServiceImpl) DeleteVessel(ctx context.Context, id int64) error {
	return s.dao.Delete(id)
}

