package handler

import (
	"backend/internal/service"
	"backend/pkg/jwt"
)

type Handlers struct {
	Auth            *AuthHandler
	Order           *OrderHandler
	Voyage          *VoyageHandler
	ShipperCompany  *ShipperCompanyHandler
	ShippingCompany *ShippingCompanyHandler
	Admin           *AdminHandler
	Port            *PortHandler
	Vessel          *VesselHandler
	ShippingLine    *ShippingLineHandler
	ImportExport    *ImportExportHandler
	Notification    *NotificationHandler
	Report          *ReportHandler
}

func NewHandlers(
	orderSvc service.OrderService,
	voyageSvc service.VoyageService,
	shipperCompanySvc service.ShipperCompanyService,
	shippingCompanySvc service.ShippingCompanyService,
	adminSvc service.AdminService,
	portSvc service.PortService,
	vesselSvc service.VesselService,
	shippingLineSvc service.ShippingLineService,
	jwtSvc jwt.JWTService,
	importExportSvc service.ImportExportService,
	notifSvc service.NotificationService,
	reportSvc service.ReportService,
) *Handlers {
	return &Handlers{
		Auth:            NewAuthHandler(shipperCompanySvc, shippingCompanySvc, adminSvc, jwtSvc),
		Order:           NewOrderHandler(orderSvc),
		Voyage:          NewVoyageHandler(voyageSvc),
		ShipperCompany:  NewShipperCompanyHandler(shipperCompanySvc),
		ShippingCompany: NewShippingCompanyHandler(shippingCompanySvc),
		Admin:           NewAdminHandler(adminSvc),
		Port:            NewPortHandler(portSvc),
		Vessel:          NewVesselHandler(vesselSvc),
		ShippingLine:    NewShippingLineHandler(shippingLineSvc),
		ImportExport:    NewImportExportHandler(importExportSvc),
		Notification:    NewNotificationHandler(notifSvc),
		Report:          NewReportHandler(reportSvc),
	}
}
