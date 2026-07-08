// Package handler 实现 HTTP 请求处理层。
//
// 层职责：
//   - 接收 HTTP 请求（解析 JSON body、query 参数、path 参数、form-data）
//   - 参数校验（通过 validator 包的 validate tag）
//   - 调用 service 层执行业务逻辑
//   - 使用 response 包构造统一 JSON 响应
//
// 分层边界：
//   - handler 不包含任何业务逻辑（不计算价格、不校验状态转换）。
//   - handler 不直接访问 DAO 或数据库（通过 service 层）。
//   - handler 只做"请求→参数→调用→响应"的转换。
//
// 依赖关系（自上而下）：
//   handler → service → biz(纯业务逻辑) + dao(数据访问)
//
// 为什么 handler 不直接调用 biz 或 dao：
//   一个完整的业务操作（如创建订单）涉及多个 DAO 操作和多个
//   biz 组件的组合，需要在 service 层编排事务。handler 如果
//   直接调用 dao 和 biz，会分散事务边界，增加耦合度。
package handler

import (
	"backend/internal/service" // service 接口层：业务编排 + 事务管理
	"backend/pkg/jwt"           // JWT 服务：token 签发/验证/刷新

	"gorm.io/gorm"
)

// Handlers 聚合所有领域 handler，作为统一入口注册到 router。
//
// 每个字段是一个领域 handler，负责一组相关的 API 接口。
// 功能速查：
//
//   Auth:
//     登录（role=shipper/shipping/admin 分流）、刷新 access_token。
//     依赖 shipperCompanySvc + shippingCompanySvc + adminSvc + jwtSvc。
//
//   Order:
//     订单 CRUD：创建（含运费计算+容量校验+事务写入）、
//     详情查询、取消（释放运力）、状态更新（状态机校验+WebSocket推送）、
//     列表分页查询（支持排序）、物流追踪（靠泊时间+起止港+船舶航线）。
//
//   Voyage:
//     航次推荐。输入起止港口+需求吨数，输出按剩余容量排序的可用航次列表。
//
//   ShipperCompany / ShippingCompany:
//     货主/船公司注册、改密、删除（admin 删除需软删除）。
//
//   Admin:
//     管理员账号创建、密码修改。
//
//   Port / Vessel / ShippingLine:
//     基础数据查询（单个详情 + 分页列表），
//     港口有按城市筛选功能，船舶有按公司筛选功能，
//     航线有解析港口序列接口。
//     Port 查询带 10 分钟缓存。
//
//   ImportExport:
//     Excel 导入导出（港口、船舶、航线、订单）。
//     导入用 excel.ReadSheet 解析上传文件，
//     导出用 excel.WriteSheet 生成 xlsx 下载。
//
//   Notification:
//     通知发送（admin 专用）、列表查询、标记已读。
//     存储使用进程内存 map，重启丢失。
//
//   Berthing:
//     靠泊记录管理：更新实际到达/出发时间。
type Handlers struct {
	Auth            *AuthHandler            // 登录、刷新 token
	Order           *OrderHandler           // 订单 CRUD + 追踪
	Voyage          *VoyageHandler          // 航次推荐
	ShipperCompany  *ShipperCompanyHandler   // 货主公司注册/改密/删除
	ShippingCompany *ShippingCompanyHandler  // 船公司注册/改密/删除
	Admin           *AdminHandler           // 管理员创建/改密
	Port            *PortHandler            // 港口查询（带缓存）
	Vessel          *VesselHandler          // 船舶查询
	ShippingLine    *ShippingLineHandler    // 航线查询 + 港口序列解析
	ImportExport    *ImportExportHandler    // Excel 导入导出
	Notification    *NotificationHandler    // 通知发送/列表/已读
	Report          *ReportHandler          // 报表统计
	Berthing        *BerthingHandler        // 靠泊记录管理
	City            *CityHandler            // 城市查询
	Cargo           *CargoHandler           // 货物管理
}

// NewHandlers 创建所有 Handler 实例，依赖 service 层和 jwt 服务。
//
// 参数是所有 service 接口，由 main 函数在初始化所有组件后传入。
// 这是典型的手动依赖注入（manual DI），没有使用任何 DI 框架。
//
// 参数说明（按用途分组）：
//   - orderSvc:           订单服务——创建/取消/状态更新/查询/追踪
//   - voyageSvc:          航次服务——运力查询/航次推荐
//   - shipperCompanySvc:  货主公司服务——注册/登录/改密/删除
//   - shippingCompanySvc: 船公司服务——注册/登录/改密/删除
//   - adminSvc:           管理员服务——创建/登录/改密
//   - portSvc:            港口服务——详情/列表/按城市筛选
//   - vesselSvc:          船舶服务——详情/列表/按公司筛选
//   - shippingLineSvc:    航线服务——详情/列表/港口序列解析
//   - jwtSvc:             JWT 服务——签发/验证/刷新令牌（用于 AuthHandler）
//   - importExportSvc:    导入导出服务——批量导入/导出 Excel
//   - notifSvc:           通知服务——发送/列表/标记已读
//   - reportSvc:          报表服务——订单统计/航次利用率
//   - berthingSvc:        靠泊服务——更新实际到达/出发时间
//   - citySvc:            城市查询服务——分页查询城市列表
func NewHandlers(
	db *gorm.DB,
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
	berthingSvc service.VoyageBerthingService,
	citySvc service.CityService,
	cargoSvc service.CargoService,
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
		ShippingLine:    NewShippingLineHandler(shippingLineSvc, db),
		ImportExport:    NewImportExportHandler(importExportSvc),
		Notification:    NewNotificationHandler(notifSvc),
		Report:          NewReportHandler(reportSvc),
		Berthing:        NewBerthingHandler(berthingSvc),
		City:            NewCityHandler(citySvc),
		Cargo:           NewCargoHandler(cargoSvc),
	}
}
