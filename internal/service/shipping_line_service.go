package service

import (
	"context"

	"backend/internal/biz"
	"backend/internal/dao"
	"backend/internal/model"
	pkgerr "backend/pkg/errors"
)

type ShippingLineService interface {
	GetLineByID(ctx context.Context, id int64) (*model.ShippingLine, error)
	ListLines(ctx context.Context, page, pageSize int) ([]model.ShippingLine, int64, error)
	ListLinesByCompany(ctx context.Context, companyID int64, page, pageSize int) ([]model.ShippingLine, int64, error)
	GetPortSequence(ctx context.Context, lineID int64) ([]int64, error)
}

type shippingLineServiceImpl struct {
	dao           dao.ShippingLineDAO
	portSeqParser biz.PortSequenceParser
}

func NewShippingLineService(dao dao.ShippingLineDAO, portSeqParser biz.PortSequenceParser) ShippingLineService {
	return &shippingLineServiceImpl{
		dao:           dao,
		portSeqParser: portSeqParser,
	}
}

func (s *shippingLineServiceImpl) GetLineByID(ctx context.Context, id int64) (*model.ShippingLine, error) {
	logger := Logger.With("method", "GetLineByID", "line_id", id)
	logger.Debug("fetching shipping line")

	line, err := s.dao.GetByID(id)
	if err != nil {
		logger.Error("shipping line not found", "error", err)
		return nil, pkgerr.NotFound("shipping line not found")
	}
	return line, nil
}

func (s *shippingLineServiceImpl) ListLines(ctx context.Context, page, pageSize int) ([]model.ShippingLine, int64, error) {
	logger := Logger.With("method", "ListLines", "page", page, "page_size", pageSize)
	logger.Debug("listing shipping lines")

	lines, total, err := s.dao.List(page, pageSize)
	if err != nil {
		logger.Error("failed to list lines", "error", err)
		return nil, 0, err
	}
	logger.Debug("lines listed", "count", len(lines), "total", total)
	return lines, total, nil
}

func (s *shippingLineServiceImpl) ListLinesByCompany(ctx context.Context, companyID int64, page, pageSize int) ([]model.ShippingLine, int64, error) {
	logger := Logger.With("method", "ListLinesByCompany", "company_id", companyID, "page", page, "page_size", pageSize)
	logger.Debug("listing shipping lines by company")

	lines, total, err := s.dao.ListByShippingCompany(companyID, page, pageSize)
	if err != nil {
		logger.Error("failed to list lines by company", "error", err)
		return nil, 0, err
	}
	logger.Debug("lines listed", "count", len(lines), "total", total)
	return lines, total, nil
}

func (s *shippingLineServiceImpl) GetPortSequence(ctx context.Context, lineID int64) ([]int64, error) {
	logger := Logger.With("method", "GetPortSequence", "line_id", lineID)
	logger.Debug("getting port sequence")

	line, err := s.dao.GetByID(lineID)
	if err != nil {
		logger.Error("shipping line not found", "error", err)
		return nil, pkgerr.NotFound("shipping line not found")
	}
	if line.PortSequence == nil {
		logger.Error("port sequence is nil")
		return nil, pkgerr.BadRequest("port sequence missing")
	}

	portIDs, err := s.portSeqParser.Parse(*line.PortSequence)
	if err != nil {
		logger.Error("failed to parse port sequence", "error", err)
		return nil, pkgerr.BadRequest("invalid port sequence")
	}

	logger.Debug("port sequence retrieved", "count", len(portIDs))
	return portIDs, nil
}
