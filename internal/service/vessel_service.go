package service

import (
	"context"

	"backend/internal/dao"
	"backend/internal/model"
	pkgerr "backend/pkg/errors"
)

type VesselService interface {
	GetVesselByID(ctx context.Context, id int64) (*model.Vessel, error)
	ListVessels(ctx context.Context, page, pageSize int) ([]model.Vessel, int64, error)
	ListVesselsByCompany(ctx context.Context, companyID int64, page, pageSize int) ([]model.Vessel, int64, error)
}

type vesselServiceImpl struct {
	dao dao.VesselDAO
}

func NewVesselService(dao dao.VesselDAO) VesselService {
	return &vesselServiceImpl{dao: dao}
}

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

func (s *vesselServiceImpl) ListVessels(ctx context.Context, page, pageSize int) ([]model.Vessel, int64, error) {
	logger := Logger.With("method", "ListVessels", "page", page, "page_size", pageSize)
	logger.Debug("listing vessels")

	vessels, total, err := s.dao.List(page, pageSize)
	if err != nil {
		logger.Error("failed to list vessels", "error", err)
		return nil, 0, err
	}
	logger.Debug("vessels listed", "count", len(vessels), "total", total)
	return vessels, total, nil
}

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
