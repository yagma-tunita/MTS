package router

import (
	"backend/internal/handler"
	"backend/net/middleware"
	"backend/net/protect"
	"backend/net/websocket"
	"backend/pkg/jwt"

	"github.com/gin-gonic/gin"
)

func Setup(h *handler.Handlers, jwtSvc jwt.JWTService) *gin.Engine {
	r := gin.New()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/ws", websocket.ServeWS(jwtSvc))

	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.NewCORS(middleware.DefaultCORSConfig()))
	r.Use(protect.SecurityHeaders())
	r.Use(protect.Honeypot(protect.DefaultHoneypotConfig()))
	r.Use(protect.IPBlocklist(protect.DefaultIPBlocklistConfig()))
	r.Use(protect.RequestGuard(protect.DefaultRequestGuardConfig()))
	r.Use(middleware.RateLimit(middleware.DefaultRateLimiterConfig()))

	api := r.Group("/api/v1")
	{
		public := api.Group("/")
		{
			public.POST("/auth/login", h.Auth.Login)
			public.POST("/auth/refresh", h.Auth.RefreshToken)
			public.POST("/shipper/register", h.ShipperCompany.Register)
			public.POST("/shipping/register", h.ShippingCompany.Register)
		}

		authMw := middleware.NewAuthMiddleware(jwtSvc)
		protected := api.Group("/")
		protected.Use(authMw.RequireAuth())
		{
			protected.POST("/shipper/password/:id", h.ShipperCompany.UpdatePassword)
			protected.POST("/shipping/password/:id", h.ShippingCompany.UpdatePassword)

			protected.POST("/orders", h.Order.CreateOrder)
			protected.GET("/orders/:id", h.Order.GetOrder)
			protected.POST("/orders/:id/cancel", h.Order.CancelOrder)
			protected.PUT("/orders/:id/status", h.Order.UpdateOrderStatus)
			protected.GET("/orders", h.Order.ListOrders)
			protected.GET("/orders/:id/tracking", h.Order.GetOrderTracking)

			protected.GET("/voyages/recommend", h.Voyage.Recommend)

			protected.GET("/ports", h.Port.ListPorts)
			protected.GET("/ports/:id", h.Port.GetPort)
			protected.GET("/vessels", h.Vessel.ListVessels)
			protected.GET("/vessels/:id", h.Vessel.GetVessel)
			protected.GET("/shipping-lines", h.ShippingLine.ListLines)
			protected.GET("/shipping-lines/:id", h.ShippingLine.GetLine)
			protected.GET("/shipping-lines/:id/port-sequence", h.ShippingLine.GetPortSequence)

			protected.GET("/export/ports", h.ImportExport.ExportPorts)
			protected.POST("/import/ports", h.ImportExport.ImportPorts)
			protected.GET("/export/vessels", h.ImportExport.ExportVessels)
			protected.POST("/import/vessels", h.ImportExport.ImportVessels)
			protected.GET("/export/shipping-lines", h.ImportExport.ExportShippingLines)
			protected.POST("/import/shipping-lines", h.ImportExport.ImportShippingLines)
			protected.GET("/export/orders", h.ImportExport.ExportOrders)

			protected.GET("/notifications", h.Notification.ListNotifications)
			protected.PUT("/notifications/:id/read", h.Notification.MarkAsRead)

			protected.GET("/reports/orders", h.Report.OrderStatistics)
			protected.GET("/reports/voyage-utilization", h.Report.VoyageUtilization)

			adminGroup := protected.Group("/admin")
			adminGroup.Use(authMw.RequireRole("admin"))
			{
				adminGroup.POST("/register", h.Admin.Create)
				adminGroup.POST("/password/:id", h.Admin.UpdatePassword)
				adminGroup.POST("/notifications", h.Notification.SendNotification)
			}
		}
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": 404, "message": "not found"})
	})
	return r
}
