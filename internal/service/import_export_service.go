package service

import (
	"context"
	"fmt"
	"time"

	"backend/internal/dao"
	"backend/internal/model"
	"backend/pkg/excel"

	"gorm.io/gorm"
)

type ImportExportService interface {
	ExportPorts(ctx context.Context) ([]byte, error)
	ImportPorts(ctx context.Context, rows [][]string) (int, error)

	ExportVessels(ctx context.Context) ([]byte, error)
	ImportVessels(ctx context.Context, rows [][]string) (int, error)

	ExportShippingLines(ctx context.Context) ([]byte, error)
	ImportShippingLines(ctx context.Context, rows [][]string) (int, error)

	ExportOrders(ctx context.Context, shipperCompanyID int64) ([]byte, error)
}

type importExportServiceImpl struct {
	db              *gorm.DB
	portDAO         dao.PortDAO
	vesselDAO       dao.VesselDAO
	shippingLineDAO dao.ShippingLineDAO
	orderDAO        dao.ShippingOrderDAO
}

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

// ExportPorts
func (s *importExportServiceImpl) ExportPorts(ctx context.Context) ([]byte, error) {
	ports, _, err := s.portDAO.List(1, 10000)
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
	return excel.WriteSheet(headers, data)
}

// ImportPorts
func (s *importExportServiceImpl) ImportPorts(ctx context.Context, rows [][]string) (int, error) {
	if len(rows) < 2 {
		return 0, fmt.Errorf("no data rows")
	}
	// Skip header row
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

// ExportVessels
func (s *importExportServiceImpl) ExportVessels(ctx context.Context) ([]byte, error) {
	vessels, _, err := s.vesselDAO.List(1, 10000)
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
	return excel.WriteSheet(headers, data)
}

// ImportVessels
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

// ExportShippingLines
func (s *importExportServiceImpl) ExportShippingLines(ctx context.Context) ([]byte, error) {
	lines, _, err := s.shippingLineDAO.List(1, 10000)
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
	return excel.WriteSheet(headers, data)
}

// ImportShippingLines
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

// ExportOrders (for a specific shipper)
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
	return excel.WriteSheet(headers, data)
}

// Helper functions for nil pointers
func nullString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
func nullInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}
func nullFloat64(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
func nullInt8(i *int8) int8 {
	if i == nil {
		return 0
	}
	return *i
}
func nullInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}
func strPtr(s string) *string       { return &s }
func int64Ptr(i int64) *int64       { return &i }
func float64Ptr(f float64) *float64 { return &f }
func int32Ptr(i int32) *int32       { return &i }
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}
