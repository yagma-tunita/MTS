// main 是 MTS（Maritime Transport System）应用的入口点。
//
// 初始化顺序（严格依赖序，不可打乱）：
//   config → logger → database → dao → biz → service → handler → router → server
//
// 每一层的组件通过手动依赖注入（构造函数传参）拼接。
// 整个 main 函数就是一个巨大的"组合根"（Composition Root）。
package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/internal/biz"
	"backend/internal/dao"
	"backend/internal/handler"
	"backend/internal/model"
	"backend/internal/notify"
	"backend/internal/service"
	"backend/net/router"
	ws "backend/net/websocket"
	"backend/pkg/config"
	"backend/pkg/database"
	"backend/pkg/jwt"
	"backend/pkg/logger"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "backend/docs"
)

// main MTS 入口。初始化顺序：config→logger→database→dao→biz→service→handler→router→server
func main() {
	// ═══════════════════════════════════════════════════════════════
	// 第 1 步：加载配置
	// ═══════════════════════════════════════════════════════════════
	// 读取 config.yaml + 环境变量覆盖。MustLoad 在失败时直接 panic，
	// 因为配置是其他所有组件的基础依赖，配置失败无法继续。
	cfg := config.MustLoad("config.yaml")
	slog.Info("config loaded", "file", "config.yaml")

	// ═══════════════════════════════════════════════════════════════
	// 第 2 步：初始化日志
	// ═══════════════════════════════════════════════════════════════
	// 使用 slog（Go 1.21 标准库结构化日志）+ lumberjack（文件轮转）。
	// Init 必须在所有日志输出之前调用，全局设置默认 logger。
	logger.Init(cfg.Log.Level, cfg.Log.Format, cfg.Log.OutputPath,
		cfg.Log.MaxSize, cfg.Log.MaxBackups, cfg.Log.MaxAge, cfg.Log.Compress)
	slog.Info("logger initialized", "level", cfg.Log.Level, "output", cfg.Log.OutputPath)

	// ═══════════════════════════════════════════════════════════════
	// 第 3 步：连接数据库
	// ═══════════════════════════════════════════════════════════════
	// 初始化 GORM 连接池。配置包括：DSN、连接数限制（MaxOpenConns/MaxIdleConns）、
	// 连接生命周期（ConnMaxLifetime/ConnMaxIdleTime）、慢查询阈值（200ms）。
	db := database.MustNewMySQL(cfg.Database, cfg.Log.Level, 200*time.Millisecond)
	defer func() {
		if err := database.Close(db); err != nil {
			slog.Error("failed to close database", "error", err)
		}
	}()
	slog.Info("database connected")

	// ═══════════════════════════════════════════════════════════════
	// 第 4 步：自动建表（可选）
	// ═══════════════════════════════════════════════════════════════
	// 环境变量 AUTO_MIGRATE=true 时启用。生产环境建议关闭，
	// 使用 sql/tables_mysql.sql 手动建表或通过 CI/CD 执行 DDL。
	if os.Getenv("AUTO_MIGRATE") == "true" {
		if err := db.AutoMigrate(
			&model.City{},
			&model.ShipperCompany{},
			&model.ShippingCompany{},
			&model.Admin{},
			&model.Port{},
			&model.Berth{},
			&model.Vessel{},
			&model.ShippingLine{},
			&model.VoyageCargoNote{},
			&model.VoyageBerthing{},
			&model.ShippingOrder{},
			&model.OrderCargo{},
			&model.SegmentCapacityUsage{},
		); err != nil {
			slog.Error("auto migration failed", "error", err)
		} else {
			slog.Info("auto migration completed")
		}
	}

	// 新表（cargo_type, line_vessel）无条件自动创建，不依赖 AUTO_MIGRATE
	if err := db.AutoMigrate(
		&model.CargoType{},
		&model.LineVessel{},
	); err != nil {
		slog.Error("auto migrate new tables failed", "error", err)
	} else {
		slog.Info("new tables auto migration completed")
	}

	// ═══════════════════════════════════════════════════════════════
	// 第 5 步：初始化 JWT 服务
	// ═══════════════════════════════════════════════════════════════
	// JWT 服务用于签发和验证 access_token / refresh_token。
	// 密钥从 config 中读取，生产环境通过 JWT_SECRET 环境变量注入。
	jwtSvc := jwt.NewJWTService(cfg.JWT.Secret, cfg.JWT.AccessExpire, cfg.JWT.RefreshExpire)

	// ═══════════════════════════════════════════════════════════════
	// 第 6 步：初始化 DAO 层（数据访问对象）
	// ═══════════════════════════════════════════════════════════════
	// DAO 封装了单表的 CRUD 操作，每个 DAO 对应一个 model/ 中的数据模型。
	// DAO 不包含业务逻辑，只做"查数据"和"写数据"。
	vesselDAO := dao.NewVesselDAO(db)
	// └─ 船舶 DAO：vessel 表的增删改查，用于获取船舶参数（DWT、名称等）
	shippingLineDAO := dao.NewShippingLineDAO(db)
	// └─ 航线 DAO：shipping_line 表的增删改查，含港口序列 JSON
	voyageCargoNoteDAO := dao.NewVoyageCargoNoteDAO(db)
	// └─ 航次货物记录 DAO：voyage_cargo_note 表的操作，含累积运力更新
	shipperCompanyDAO := dao.NewShipperCompanyDAO(db)
	// └─ 货主公司 DAO：shipper_company 表，用于登录认证和账号管理
	shippingCompanyDAO := dao.NewShippingCompanyDAO(db)
	// └─ 船公司 DAO：shipping_company 表，用于登录认证
	orderDAO := dao.NewShippingOrderDAO(db)
	// └─ 订单 DAO：shipping_order 表的核心 CRUD
	orderCargoDAO := dao.NewOrderCargoDAO(db)
	// └─ 订单货物 DAO：order_cargo 表（订单的货物明细）
	segmentUsageDAO := dao.NewSegmentCapacityUsageDAO(db)
	// └─ 航段容量占用 DAO：segment_capacity_usage 表，记录每个订单在每个航段占用的吨位
	adminDAO := dao.NewAdminDAO(db)
	// └─ 管理员 DAO：admin 表
	portDAO := dao.NewPortDAO(db)
	// └─ 港口 DAO：port 表，含城市关联
	cityDAO := dao.NewCityDAO(db)
	// └─ 城市 DAO：city 表
	berthingDAO := dao.NewVoyageBerthingDAO(db)
	// └─ 靠泊 DAO：voyage_berthing 表，含靠泊计划/实际时间

	// ═══════════════════════════════════════════════════════════════
	// 第 7 步：初始化 Biz 层（纯业务逻辑组件）
	// ═══════════════════════════════════════════════════════════════
	// biz 组件不依赖任何 I/O，只做数学计算或规则校验。
	// 以下是各个组件的功能说明：
	//   - PortSequenceParser: 将 JSON 字符串 "[1,2,3]" 解析为 []int64{1,2,3}
	//   - SegmentCalculator:  根据港口序列和起止港，计算途径的所有邻接航段
	//     (如 [1,2,3,5] 中从 1 到 5，返回 (1,2),(2,3),(3,5))
	//   - CapacityChecker:   校验每个航段是否容纳得下订单的货物重量
	//     （已占用 + 新货物 ≤ 最大载重）
	//   - CostCalculator:    汇总多个 CargoItem 的总重量、总体积、总费用
	//   - OrderNoGenerator:  生成订单号 ORD20260706xxxxxxxx（日期+随机hex）
	//   - OrderStateMachine: 订单状态转换规则校验（草稿→确认→运输中→完成/取消）
	//   - VoyageRecommender: 推荐引擎，遍历所有航次按剩余容量排序
	bizContainer := biz.NewBizContainer()

	// ═══════════════════════════════════════════════════════════════
	// 第 8 步：初始化 Service 层
	// ═══════════════════════════════════════════════════════════════
	// Service 层编排业务流程，组合多个 DAO 和 biz 组件完成业务操作。
	// 每个 Service 的构造函数参数清晰地表达了它的依赖关系。

	// WebSocket 推送服务（向用户实时推送订单状态变更）
	wsSvc := service.NewWebSocketService()

	// 订单服务（最核心的 Service）：
	// 依赖：6 个 DAO（vessel, shippingLine, voyageCargoNote, order, orderCargo, segmentUsage）
	//      + 5 个 biz 组件（PortSequenceParser, SegmentCalculator, CapacityChecker, CostCalculator, OrderNoGenerator, OrderStateMachine）
	//      + WebSocket 服务
	orderSvc := service.NewOrderService(
		db, orderDAO, orderCargoDAO, segmentUsageDAO, voyageCargoNoteDAO,
		vesselDAO, shippingLineDAO,
		bizContainer.PortSequenceParser, bizContainer.SegmentCalculator,
		bizContainer.CapacityChecker, bizContainer.CostCalculator,
		bizContainer.OrderNoGenerator, bizContainer.OrderStateMachine,
		wsSvc,
	)

	// 航次服务（运力查询 + 推荐引擎）：
	// 依赖：4 个 DAO + Biz 中的 PortSequenceParser 和 VoyageRecommender
	voyageSvc := service.NewVoyageService(
		db, shippingLineDAO, vesselDAO, voyageCargoNoteDAO, segmentUsageDAO,
		bizContainer.PortSequenceParser, bizContainer.VoyageRecommender,
	)

	// 公司服务（注册、登录、改密、删除）：
	shipperCompanySvc := service.NewShipperCompanyService(shipperCompanyDAO)
	shippingCompanySvc := service.NewShippingCompanyService(shippingCompanyDAO)

	// 管理员服务：
	adminSvc := service.NewAdminService(adminDAO)

	// 基础数据查询服务（带缓存）：
	citySvc := service.NewCityService(cityDAO)
	// └─ 城市查询（分页列表）
	portSvc := service.NewPortService(portDAO)
	// └─ 港口查询（支持按城市筛选。GetByID 和 List 有 10 分钟缓存）
	vesselSvc := service.NewVesselService(vesselDAO)
	// └─ 船舶查询
	shippingLineSvc := service.NewShippingLineService(shippingLineDAO, bizContainer.PortSequenceParser)
	// └─ 航线查询 + 港口序列解析

	// Excel 导入导出服务：
	importExportSvc := service.NewImportExportService(db, portDAO, vesselDAO, shippingLineDAO, orderDAO)

	// 通知服务（内存存储 + 可选邮件/SMS 发送）：
	notifyProv := notify.NewProvider(
		notify.EmailConfig{
			SMTPHost: cfg.Notify.Email.SMTPHost,
			SMTPPort: cfg.Notify.Email.SMTPPort,
			Username: cfg.Notify.Email.Username,
			Password: cfg.Notify.Email.Password,
			FromAddr: cfg.Notify.Email.FromAddr,
			FromName: cfg.Notify.Email.FromName,
		},
		notify.SMSConfig{
			Provider:        cfg.Notify.SMS.Provider,
			AccessKeyID:     cfg.Notify.SMS.AccessKeyID,
			AccessKeySecret: cfg.Notify.SMS.AccessKeySecret,
			SignName:        cfg.Notify.SMS.SignName,
			TemplateCode:    cfg.Notify.SMS.TemplateCode,
		},
	)
	notifSvc := service.NewNotificationService(notifyProv)

	// 靠泊服务（更新实际到达/出发时间）：
	berthingSvc := service.NewVoyageBerthingService(berthingDAO)

	// 货物类型 DAO + Service：
	cargoTypeDAO := dao.NewCargoTypeDAO(db)
	cargoTypeSvc := service.NewCargoTypeService(cargoTypeDAO)

	// 货物查询服务（admin 查看所有货物）：
	cargoSvc := service.NewCargoService(orderCargoDAO)

	// 报表统计服务：
	reportSvc := service.NewReportService(db)

	// ═══════════════════════════════════════════════════════════════
	// 第 9 步：初始化 Handler 层
	// ═══════════════════════════════════════════════════════════════
	// Handler 负责解析 HTTP 请求、调用 Service、返回 JSON 响应。
	// Handlers 是聚合结构体，包含所有领域 handler，传给路由注册。
	handlers := handler.NewHandlers(
		db, orderSvc, voyageSvc, shipperCompanySvc, shippingCompanySvc,
		adminSvc, portSvc, vesselSvc, shippingLineSvc, jwtSvc, importExportSvc,
		notifSvc, reportSvc, berthingSvc, citySvc, cargoSvc, cargoTypeSvc,
	)

	// ═══════════════════════════════════════════════════════════════
	// 第 10 步：配置路由
	// ═══════════════════════════════════════════════════════════════
	// Setup 创建 Gin 引擎，注册全局中间件（Log/Recovery/CORS/Security/Honeypot/RateLimit）、
	// 公开路由（login/register）和受保护路由（所有业务接口 + admin 专用接口）。
	r := router.Setup(handlers, jwtSvc)

	// Swagger API 文档（自动生成的交互式文档）
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// pprof 性能分析端点（环境变量 ENABLE_PPROF=true 时启用）
	if os.Getenv("ENABLE_PPROF") == "true" {
		pprofGroup := r.Group("/debug/pprof")
		{
			pprofGroup.GET("/", gin.WrapH(http.HandlerFunc(pprof.Index)))
			pprofGroup.GET("/cmdline", gin.WrapH(http.HandlerFunc(pprof.Cmdline)))
			pprofGroup.GET("/profile", gin.WrapH(http.HandlerFunc(pprof.Profile)))
			pprofGroup.GET("/symbol", gin.WrapH(http.HandlerFunc(pprof.Symbol)))
			pprofGroup.GET("/trace", gin.WrapH(http.HandlerFunc(pprof.Trace)))
		}
		slog.Info("pprof enabled at /debug/pprof")
	}

	// ═══════════════════════════════════════════════════════════════
	// 第 11 步：启动 HTTP 服务器
	// ═══════════════════════════════════════════════════════════════
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		slog.Info("server started", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 后台 goroutine：每 30 秒输出数据库连接池状态（用于监控和调优）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			sqlDB, err := db.DB()
			if err == nil {
				stats := sqlDB.Stats()
				slog.Debug("db connection pool",
					"max_open", stats.MaxOpenConnections,
					"open", stats.OpenConnections,
					"in_use", stats.InUse,
					"idle", stats.Idle,
					"wait_count", stats.WaitCount,
				)
			}
		}
	}()

	// ═══════════════════════════════════════════════════════════════
	// 第 12 步：优雅关闭
	// ═══════════════════════════════════════════════════════════════
	// 收到 SIGINT（Ctrl+C）或 SIGTERM（K8s 停止 Pod）时触发。
	// 关闭顺序（不可颠倒）：
	//   1. 先关 WebSocket Hub —— 停止推送新消息
	//   2. 再关 HTTP Server —— 停止接受新请求，等待旧请求完成（5 秒超时）
	<-quit
	slog.Info("shutting down server...")

	ws.ShutdownHub()
	slog.Info("WebSocket hub stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("server exited")
}
