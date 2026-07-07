package service

import (
	"context"

	"backend/internal/biz"
	"backend/internal/dao"
	"backend/internal/model"
	pkgerr "backend/pkg/errors"
)

// ShippingLineService 航线服务接口，支持查询单个、列表、获取港口序列、增删改。
type ShippingLineService interface {
	GetLineByID(ctx context.Context, id int64) (*model.ShippingLine, error)
	ListLines(ctx context.Context, page, pageSize int) ([]model.ShippingLine, int64, error)
	ListLinesByCompany(ctx context.Context, companyID int64, page, pageSize int) ([]model.ShippingLine, int64, error)
	GetPortSequence(ctx context.Context, lineID int64) ([]int64, error)
	CreateLine(ctx context.Context, line *model.ShippingLine) error
	UpdateLine(ctx context.Context, line *model.ShippingLine) error
	DeleteLine(ctx context.Context, id int64) error
}

// shippingLineServiceImpl 是 ShippingLineService 接口的私有实现。
type shippingLineServiceImpl struct {
	dao           dao.ShippingLineDAO
	portSeqParser biz.PortSequenceParser
}

// NewShippingLineService 创建航线服务实例
func NewShippingLineService(dao dao.ShippingLineDAO, portSeqParser biz.PortSequenceParser) ShippingLineService {
	return &shippingLineServiceImpl{
		dao:           dao,
		portSeqParser: portSeqParser,
	}
}

// GetLineByID 查询航线详情。
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

// ListLines 分页查询航线列表。
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

// ListLinesByCompany 根据船公司 ID 分页查询该公司航线。
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

// GetPortSequence 解析航线的港口序列 JSON，返回 int64 切片。
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

// CreateLine 创建新航线。
func (s *shippingLineServiceImpl) CreateLine(ctx context.Context, line *model.ShippingLine) error {
	return s.dao.Create(line)
}

// UpdateLine 更新航线信息。
func (s *shippingLineServiceImpl) UpdateLine(ctx context.Context, line *model.ShippingLine) error {
	return s.dao.Update(line)
}

// DeleteLine 软删除航线。
func (s *shippingLineServiceImpl) DeleteLine(ctx context.Context, id int64) error {
	return s.dao.Delete(id)
}

