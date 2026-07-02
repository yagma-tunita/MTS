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

func main() {
	cfg := config.MustLoad("config.yaml")
	slog.Info("config loaded", "file", "config.yaml")

	logger.Init(cfg.Log.Level, cfg.Log.Format, cfg.Log.OutputPath,
		cfg.Log.MaxSize, cfg.Log.MaxBackups, cfg.Log.MaxAge, cfg.Log.Compress)
	slog.Info("logger initialized", "level", cfg.Log.Level, "output", cfg.Log.OutputPath)

	db := database.MustNewMySQL(cfg.Database, cfg.Log.Level, 200*time.Millisecond)
	defer func() {
		if err := database.Close(db); err != nil {
			slog.Error("failed to close database", "error", err)
		}
	}()
	slog.Info("database connected")

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

	jwtSvc := jwt.NewJWTService(cfg.JWT.Secret, cfg.JWT.AccessExpire, cfg.JWT.RefreshExpire)

	vesselDAO := dao.NewVesselDAO(db)
	shippingLineDAO := dao.NewShippingLineDAO(db)
	voyageCargoNoteDAO := dao.NewVoyageCargoNoteDAO(db)
	shipperCompanyDAO := dao.NewShipperCompanyDAO(db)
	shippingCompanyDAO := dao.NewShippingCompanyDAO(db)
	orderDAO := dao.NewShippingOrderDAO(db)
	orderCargoDAO := dao.NewOrderCargoDAO(db)
	segmentUsageDAO := dao.NewSegmentCapacityUsageDAO(db)
	adminDAO := dao.NewAdminDAO(db)
	portDAO := dao.NewPortDAO(db)

	bizContainer := biz.NewBizContainer()

	wsSvc := service.NewWebSocketService()

	orderSvc := service.NewOrderService(
		db, orderDAO, orderCargoDAO, segmentUsageDAO, voyageCargoNoteDAO,
		vesselDAO, shippingLineDAO,
		bizContainer.PortSequenceParser, bizContainer.SegmentCalculator,
		bizContainer.CapacityChecker, bizContainer.CostCalculator,
		bizContainer.OrderNoGenerator, bizContainer.OrderStateMachine,
		wsSvc,
	)

	voyageSvc := service.NewVoyageService(
		db, shippingLineDAO, vesselDAO, voyageCargoNoteDAO, segmentUsageDAO,
		bizContainer.PortSequenceParser, bizContainer.VoyageRecommender,
	)

	shipperCompanySvc := service.NewShipperCompanyService(shipperCompanyDAO)
	shippingCompanySvc := service.NewShippingCompanyService(shippingCompanyDAO)
	adminSvc := service.NewAdminService(adminDAO)
	portSvc := service.NewPortService(portDAO)
	vesselSvc := service.NewVesselService(vesselDAO)
	shippingLineSvc := service.NewShippingLineService(shippingLineDAO, bizContainer.PortSequenceParser)

	importExportSvc := service.NewImportExportService(db, portDAO, vesselDAO, shippingLineDAO, orderDAO)

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
	reportSvc := service.NewReportService(db)

	handlers := handler.NewHandlers(
		orderSvc, voyageSvc, shipperCompanySvc, shippingCompanySvc,
		adminSvc, portSvc, vesselSvc, shippingLineSvc, jwtSvc, importExportSvc,
		notifSvc, reportSvc,
	)

	r := router.Setup(handlers, jwtSvc)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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
