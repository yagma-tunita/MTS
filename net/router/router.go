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

	// Health check endpoint (no middleware)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// WebSocket endpoint - placed before any middlewares to avoid interference
	r.GET("/ws", websocket.ServeWS(jwtSvc))

	// Global middlewares (order matters)
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
		// Public routes (no authentication)
		public := api.Group("/")
		{
			public.POST("/auth/login", h.Auth.Login)
			public.POST("/auth/refresh", h.Auth.RefreshToken)
			public.POST("/shipper/register", h.ShipperCompany.Register)
			public.POST("/shipping/register", h.ShippingCompany.Register)
		}

		// Protected routes (require JWT)
		protected := api.Group("/")
		protected.Use(middleware.NewAuthMiddleware(jwtSvc).RequireAuth())
		{
			// Password updates
			protected.POST("/shipper/password", h.ShipperCompany.UpdatePassword)
			protected.POST("/shipping/password", h.ShippingCompany.UpdatePassword)

			// Orders
			protected.POST("/orders", h.Order.CreateOrder)
			protected.GET("/orders/:id", h.Order.GetOrder)
			protected.POST("/orders/:id/cancel", h.Order.CancelOrder)
			protected.PUT("/orders/:id/status", h.Order.UpdateOrderStatus)
			protected.GET("/orders", h.Order.ListOrders)
			protected.GET("/orders/:id/tracking", h.Order.GetOrderTracking)

			// Voyage recommendation
			protected.GET("/voyages/recommend", h.Voyage.Recommend)

			// Basic data
			protected.GET("/ports", h.Port.ListPorts)
			protected.GET("/ports/:id", h.Port.GetPort)
			protected.GET("/vessels", h.Vessel.ListVessels)
			protected.GET("/vessels/:id", h.Vessel.GetVessel)
			protected.GET("/shipping-lines", h.ShippingLine.ListLines)
			protected.GET("/shipping-lines/:id", h.ShippingLine.GetLine)
			protected.GET("/shipping-lines/:id/port-sequence", h.ShippingLine.GetPortSequence)

			// Import/Export
			protected.GET("/export/ports", h.ImportExport.ExportPorts)
			protected.POST("/import/ports", h.ImportExport.ImportPorts)
			protected.GET("/export/vessels", h.ImportExport.ExportVessels)
			protected.POST("/import/vessels", h.ImportExport.ImportVessels)
			protected.GET("/export/shipping-lines", h.ImportExport.ExportShippingLines)
			protected.POST("/import/shipping-lines", h.ImportExport.ImportShippingLines)
			protected.GET("/export/orders", h.ImportExport.ExportOrders)

			// Notifications
			protected.GET("/notifications", h.Notification.ListNotifications)
			protected.PUT("/notifications/:id/read", h.Notification.MarkAsRead)

			// Reports
			protected.GET("/reports/orders", h.Report.OrderStatistics)
			protected.GET("/reports/voyage-utilization", h.Report.VoyageUtilization)

			// Admin only routes (require role=admin)
			adminGroup := protected.Group("/admin")
			adminGroup.Use(middleware.NewAuthMiddleware(jwtSvc).RequireRole("admin"))
			{
				adminGroup.POST("/register", h.Admin.Create)
				adminGroup.POST("/password", h.Admin.UpdatePassword)
				adminGroup.POST("/notifications", h.Notification.SendNotification)
			}
		}
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": 404, "message": "not found"})
	})
	return r
}
