package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"backend/internal/dao"
	"backend/internal/model"
	"backend/pkg/excel"

	"gorm.io/gorm"
)

// ImportExportService 导入导出服务接口，支持港口、船舶、航线、订单的 Excel 导入导出。
type ImportExportService interface {
	ExportPorts(ctx context.Context) ([]byte, error)
	ImportPorts(ctx context.Context, rows [][]string) (int, error)

	ExportVessels(ctx context.Context) ([]byte, error)
	ImportVessels(ctx context.Context, rows [][]string) (int, error)

	ExportShippingLines(ctx context.Context) ([]byte, error)
	ImportShippingLines(ctx context.Context, rows [][]string) (int, error)

	ExportOrders(ctx context.Context, shipperCompanyID int64) ([]byte, error)
}

// importExportServiceImpl 是 ImportExportService 接口的私有实现。
type importExportServiceImpl struct {
	db              *gorm.DB
	portDAO         dao.PortDAO
	vesselDAO       dao.VesselDAO
	shippingLineDAO dao.ShippingLineDAO
	orderDAO        dao.ShippingOrderDAO
}

// NewImportExportService 创建导入导出服务实例。
//
// 依赖说明：
//   db              — *gorm.DB 实例（当前未直接使用，保留以备将来扩展事务需求）
//   portDAO         — 港口 DAO：导出时 List() 全量港口，导入时 Create() 逐条写入
//   vesselDAO       — 船舶 DAO：导出时 List() 全量船舶，导入时 Create() 逐条写入
//   shippingLineDAO — 航线 DAO：导出时 List() 全量航线，导入时 Create() 逐条写入
//   orderDAO        — 订单 DAO：导出时 ListByShipper() 按货主查询订单用于导出
func NewImportExportService(
	db *gorm.DB,
	portDAO dao.PortDAO,
	vesselDAO dao.VesselDAO,
	shippingLineDAO dao.ShippingLineDAO,
	orderDAO dao.ShippingOrderDAO,
) ImportExportService {
	return &importExportServiceImpl{
		db:              db,
		portDAO:         portDAO,
		vesselDAO:       vesselDAO,
		shippingLineDAO: shippingLineDAO,
		orderDAO:        orderDAO,
	}
}

// saveExcel 将 Excel 字节流保存到本地文件（路径：./excel/mts_YYYYMMDD_type.xlsx）。
func saveExcel(data []byte, suffix string) error {
	dir := filepath.Join(".", "excel")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create excel dir: %w", err)
	}
	name := fmt.Sprintf("mts_%s_%s.xlsx", time.Now().Format("20060102"), suffix)
	return os.WriteFile(filepath.Join(dir, name), data, 0644)
}

// ExportPorts 导出所有港口为 Excel。
func (s *importExportServiceImpl) ExportPorts(ctx context.Context) ([]byte, error) {
	ports, _, err := s.portDAO.List(1, 10000, "")
	if err != nil {
		return nil, err
	}
	headers := []string{"ID", "PortName", "PortCode", "CityID", "Latitude", "Longitude", "PortType", "MaxDraftMeter"}
	data := make([][]string, len(ports))
	for i, p := range ports {
		data[i] = []string{
			fmt.Sprintf("%d", p.PortID),
			p.PortName,
			nullString(p.PortCode),
			fmt.Sprintf("%d", nullInt64(p.CityID)),
			fmt.Sprintf("%f", nullFloat64(p.Latitude)),
			fmt.Sprintf("%f", nullFloat64(p.Longitude)),
			nullString(p.PortType),
			fmt.Sprintf("%f", nullFloat64(p.MaxDraftMeter)),
		}
	}
	bytes, err := excel.WriteSheet(headers, data)
	if err != nil {
		return nil, err
	}
	saveExcel(bytes, "ports")
	return bytes, nil
}

// ImportPorts 从 Excel 行数据批量导入港口。
func (s *importExportServiceImpl) ImportPorts(ctx context.Context, rows [][]string) (int, error) {
	if len(rows) < 2 {
		return 0, fmt.Errorf("no data rows")
	}
	imported := 0
	for idx, row := range rows[1:] {
		if len(row) < 8 {
			continue
		}
		port := &model.Port{
			PortName:      row[1],
			PortCode:      strPtr(row[2]),
			CityID:        int64Ptr(excel.ParseInt(row[3])),
			Latitude:      float64Ptr(excel.ParseFloat(row[4])),
			Longitude:     float64Ptr(excel.ParseFloat(row[5])),
			PortType:      strPtr(row[6]),
			MaxDraftMeter: float64Ptr(excel.ParseFloat(row[7])),
		}
		if err := s.portDAO.Create(port); err != nil {
			return imported, fmt.Errorf("row %d: %w", idx+2, err)
		}
		imported++
	}
	return imported, nil
}

// ExportVessels 导出所有船舶为 Excel。
func (s *importExportServiceImpl) ExportVessels(ctx context.Context) ([]byte, error) {
	vessels, _, err := s.vesselDAO.List(1, 10000, "")
	if err != nil {
		return nil, err
	}
	headers := []string{"ID", "VesselName", "CallSign", "IMO", "VesselType", "MaxDeadweightTon", "GrossTonnage", "NetTonnage", "DraftMeter", "SpeedKnot", "ContainerTEU", "IsAvailable", "ShippingCompanyID"}
	data := make([][]string, len(vessels))
	for i, v := range vessels {
		data[i] = []string{
			fmt.Sprintf("%d", v.VesselID),
			v.VesselName,
			nullString(v.CallSign),
			v.IMONumber,
			nullString(v.VesselType),
			fmt.Sprintf("%f", nullFloat64(v.MaxDeadweightTon)),
			fmt.Sprintf("%f", nullFloat64(v.GrossTonnage)),
			fmt.Sprintf("%f", nullFloat64(v.NetTonnage)),
			fmt.Sprintf("%f", nullFloat64(v.DraftMeter)),
			fmt.Sprintf("%f", nullFloat64(v.SpeedKnot)),
			fmt.Sprintf("%d", nullInt32(v.ContainerTEU)),
			fmt.Sprintf("%d", v.IsAvailable),
			fmt.Sprintf("%d", nullInt64(v.ShippingCompanyID)),
		}
	}
	bytes, err := excel.WriteSheet(headers, data)
	if err != nil {
		return nil, err
	}
	saveExcel(bytes, "vessels")
	return bytes, nil
}

// ImportVessels 从 Excel 行数据批量导入船舶。
func (s *importExportServiceImpl) ImportVessels(ctx context.Context, rows [][]string) (int, error) {
	if len(rows) < 2 {
		return 0, fmt.Errorf("no data rows")
	}
	imported := 0
	for idx, row := range rows[1:] {
		if len(row) < 13 {
			continue
		}
		vessel := &model.Vessel{
			VesselName:        row[1],
			CallSign:          strPtr(row[2]),
			IMONumber:         row[3],
			VesselType:        strPtr(row[4]),
			MaxDeadweightTon:  float64Ptr(excel.ParseFloat(row[5])),
			GrossTonnage:      float64Ptr(excel.ParseFloat(row[6])),
			NetTonnage:        float64Ptr(excel.ParseFloat(row[7])),
			DraftMeter:        float64Ptr(excel.ParseFloat(row[8])),
			SpeedKnot:         float64Ptr(excel.ParseFloat(row[9])),
			ContainerTEU:      int32Ptr(int32(excel.ParseInt(row[10]))),
			IsAvailable:       int8(excel.ParseInt(row[11])),
			ShippingCompanyID: int64Ptr(excel.ParseInt(row[12])),
		}
		if err := s.vesselDAO.Create(vessel); err != nil {
			return imported, fmt.Errorf("row %d: %w", idx+2, err)
		}
		imported++
	}
	return imported, nil
}

// ExportShippingLines 导出所有航线为 Excel。
func (s *importExportServiceImpl) ExportShippingLines(ctx context.Context) ([]byte, error) {
	lines, _, err := s.shippingLineDAO.List(1, 10000, "")
	if err != nil {
		return nil, err
	}
	headers := []string{"ID", "LineName", "ShippingCompanyID", "PortSequence", "TotalDistanceNm", "DeparturePortName", "DestinationPortName", "Description"}
	data := make([][]string, len(lines))
	for i, l := range lines {
		data[i] = []string{
			fmt.Sprintf("%d", l.LineID),
			l.LineName,
			fmt.Sprintf("%d", nullInt64(l.ShippingCompanyID)),
			nullString(l.PortSequence),
			fmt.Sprintf("%f", nullFloat64(l.TotalDistanceNm)),
			nullString(l.DeparturePortName),
			nullString(l.DestinationPortName),
			nullString(l.Description),
		}
	}
	bytes, err := excel.WriteSheet(headers, data)
	if err != nil {
		return nil, err
	}
	saveExcel(bytes, "shipping_lines")
	return bytes, nil
}

// ImportShippingLines 从 Excel 行数据批量导入航线。
func (s *importExportServiceImpl) ImportShippingLines(ctx context.Context, rows [][]string) (int, error) {
	if len(rows) < 2 {
		return 0, fmt.Errorf("no data rows")
	}
	imported := 0
	for idx, row := range rows[1:] {
		if len(row) < 8 {
			continue
		}
		line := &model.ShippingLine{
			LineName:            row[1],
			ShippingCompanyID:   int64Ptr(excel.ParseInt(row[2])),
			PortSequence:        strPtr(row[3]),
			TotalDistanceNm:     float64Ptr(excel.ParseFloat(row[4])),
			DeparturePortName:   strPtr(row[5]),
			DestinationPortName: strPtr(row[6]),
			Description:         strPtr(row[7]),
		}
		if err := s.shippingLineDAO.Create(line); err != nil {
			return imported, fmt.Errorf("row %d: %w", idx+2, err)
		}
		imported++
	}
	return imported, nil
}

// ExportOrders 根据货主公司 ID 导出订单为 Excel。
func (s *importExportServiceImpl) ExportOrders(ctx context.Context, shipperCompanyID int64) ([]byte, error) {
	orders, _, err := s.orderDAO.ListByShipper(shipperCompanyID, 1, 10000)
	if err != nil {
		return nil, err
	}
	headers := []string{"OrderID", "OrderNo", "ShipperCompanyID", "CityID", "LoadNoteID", "UnloadNoteID", "DeparturePortID", "DestinationPortID", "ExpectedDepartureDate", "ExpectedArrivalDate", "TotalCost", "PaymentStatus", "OrderStatus", "TotalWeightTon", "TotalVolumeCubicMeter", "CreateTime"}
	data := make([][]string, len(orders))
	for i, o := range orders {
		data[i] = []string{
			fmt.Sprintf("%d", o.OrderID),
			o.OrderNo,
			fmt.Sprintf("%d", nullInt64(o.ShipperCompanyID)),
			fmt.Sprintf("%d", nullInt64(o.CityID)),
			fmt.Sprintf("%d", nullInt64(o.LoadNoteID)),
			fmt.Sprintf("%d", nullInt64(o.UnloadNoteID)),
			fmt.Sprintf("%d", nullInt64(o.DeparturePortID)),
			fmt.Sprintf("%d", nullInt64(o.DestinationPortID)),
			formatTimePtr(o.ExpectedDepartureDate),
			formatTimePtr(o.ExpectedArrivalDate),
			fmt.Sprintf("%f", nullFloat64(o.TotalCost)),
			fmt.Sprintf("%d", nullInt8(o.PaymentStatus)),
			fmt.Sprintf("%d", nullInt8(o.OrderStatus)),
			fmt.Sprintf("%f", nullFloat64(o.TotalWeightTon)),
			fmt.Sprintf("%f", nullFloat64(o.TotalVolumeCubicMeter)),
			o.CreateTime.Format("2006-01-02 15:04:05"),
		}
	}
	bytes, err := excel.WriteSheet(headers, data)
	if err != nil {
		return nil, err
	}
	saveExcel(bytes, "orders")
	return bytes, nil
}

// 以下为处理 nil 指针的辅助函数
func nullString(s *string) string {
	if s == nil { return "" }
	return *s
}
// nullInt64 返回 int64 指针的值，nil 时返回 0
func nullInt64(i *int64) int64 {
	if i == nil { return 0 }
	return *i
}
// nullFloat64 返回 float64 指针的值，nil 时返回 0
func nullFloat64(f *float64) float64 {
	if f == nil { return 0 }
	return *f
}
// nullInt8 返回 int8 指针的值，nil 时返回 0
func nullInt8(i *int8) int8 {
	if i == nil { return 0 }
	return *i
}
// nullInt32 返回 int32 指针的值，nil 时返回 0
func nullInt32(i *int32) int32 {
	if i == nil { return 0 }
	return *i
}
// strPtr 返回字符串指针
func strPtr(s string) *string       { return &s }
// int64Ptr int64 指针
func int64Ptr(i int64) *int64       { return &i }
// float64Ptr 返回 float64 指针
func float64Ptr(f float64) *float64 { return &f }
// int32Ptr int32 指针
func int32Ptr(i int32) *int32       { return &i }
// formatTimePtr 格式化时间指针为日期字符串
func formatTimePtr(t *time.Time) string {
	if t == nil { return "" }
	return t.Format("2006-01-02")
}


