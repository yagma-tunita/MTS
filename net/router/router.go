// 配置所有 HTTP 路由和全局中间件。
// 中间件应用顺序（由 Gin 的 Use 调用顺序决定，先调用的在外层）：
//
//	Logger → Recovery → CORS → SecurityHeaders → Honeypot
//	→ IPBlocklist → RequestGuard → RateLimit
//
// 为什么是这个顺序（详见各个中间件的注释）：
//   - Logger 在最外层：记录所有请求（包括被后续中间件拒绝的）。
//   - Recovery 紧随其后：兜住所有 panic。
//   - CORS/SecurityHeaders：修改响应头，不拦截请求。
//   - Honeypot/IPBlocklist：低成本过滤已知恶意请求。
//   - RequestGuard/RateLimit：较重量级的安全检查放最后。
//
// 路由结构概览：
//
//	/health                          GET     健康检查（无中间件）
//	/ws                              GET     WebSocket（仅 JWT 验证，无中间件）
//	/api/v1/auth/login               POST    登录（公开）
//	/api/v1/auth/refresh             POST    刷新 token（公开）
//	/api/v1/shipper/register         POST    货主注册（公开）
//	/api/v1/shipping/register        POST    船公司注册（公开）
//	/api/v1/{resources}              *       受保护接口（需 JWT）
//	/api/v1/admin/*                  *       管理员接口（需 JWT + admin 角色）
//	/api/v1/shipping-companies       GET     货主查看船公司列表（需 JWT）
//	/api/v1/berthings/:id/actual-times PUT  更新靠泊实际时间（需 JWT）
//
// 关键设计决策：
//   - WebSocket 路径 /ws 在全局中间件之前注册，避免中间件干扰。
//   - /api/v1 分组下，公开路由和受保护路由通过不同的 Gin Group 隔离。
//   - admin 子路由额外叠加 RequireRole("admin") 中间件。
package router

import (
	"backend/internal/handler"
	"backend/net/middleware"
	"backend/net/protect"
	"backend/net/websocket"
	"backend/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// Setup 创建并配置 Gin 引擎，注册所有路由和中间件。
//
// 参数 h 包含所有 handler 的聚合结构体，由 main 函数构造。
// 参数 jwtSvc 用于认证中间件。
//
// 返回值是已完全配置的 *gin.Engine，main 函数用它启动 HTTP 服务。
func Setup(h *handler.Handlers, jwtSvc jwt.JWTService) *gin.Engine {
	r := gin.New()

	// ── 健康检查 ──
	// 无任何中间件（包括 Logger），因为负载均衡器发起的健康检查
	// 频率很高（可能每秒一次），不需要记录日志。
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// ── WebSocket ──
	// 放在全局中间件之前，避免被 Logger/Recovery/RateLimit 等干扰。
	// WebSocket 的 HTTP 升级握手是特殊请求，经过某些中间件
	// 可能导致升级失败或异常行为。
	// 注意：JWT 认证是通过 URL query 参数（?token=xxx）进行的，
	// 而不是通过 Authorization header，因此不会走认证中间件。
	r.GET("/ws", websocket.ServeWS(jwtSvc))

	// ── 全局中间件 ──
	// r.Use 只对注册之后的路由生效，所以 /health 和 /ws 不受影响。
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.NewCORS(middleware.DefaultCORSConfig()))
	r.Use(protect.SecurityHeaders())
	r.Use(protect.Honeypot(protect.DefaultHoneypotConfig()))
	r.Use(protect.IPBlocklist(protect.DefaultIPBlocklistConfig()))
	r.Use(protect.RequestGuard(protect.DefaultRequestGuardConfig()))
	r.Use(middleware.RateLimit(middleware.DefaultRateLimiterConfig()))

	// ── API v1 路由组 ──
	api := r.Group("/api/v1")
	{
		// 公开接口（无需 JWT 认证）
		public := api.Group("/")
		{
			public.POST("/auth/login", h.Auth.Login)
			public.POST("/auth/refresh", h.Auth.RefreshToken)
			public.POST("/shipper/register", h.ShipperCompany.Register)
			public.POST("/shipping/register", h.ShippingCompany.Register)
		}

		// 受保护接口（需 JWT 认证）
		authMw := middleware.NewAuthMiddleware(jwtSvc)
		protected := api.Group("/")
		protected.Use(authMw.RequireAuth())
		{
			// 当前登录用户信息
			protected.GET("/auth/me", h.Auth.Me)

			// 密码管理（shipper/shipping 均需登录后修改自己的密码）
			protected.POST("/shipper/password/:id", h.ShipperCompany.UpdatePassword)
			protected.POST("/shipping/password/:id", h.ShippingCompany.UpdatePassword)

			// 订单管理 CRUD
			protected.POST("/orders", h.Order.CreateOrder)
			protected.GET("/orders/:id", h.Order.GetOrder)
			protected.POST("/orders/:id/cancel", h.Order.CancelOrder)
			protected.PUT("/orders/:id/status", h.Order.UpdateOrderStatus)
			protected.GET("/orders", h.Order.ListOrders)
			protected.GET("/orders/:id/tracking", h.Order.GetOrderTracking)

			// 运力推荐（核心业务接口）
			protected.GET("/voyages/recommend", h.Voyage.Recommend)
			// 航线申请（船公司创建航次靠泊记录）
			protected.POST("/voyages/berthing", h.Voyage.CreateVoyageBerthing)

			// 基础数据查询（分页列表 + 单条详情）
			protected.GET("/cities", h.City.ListCities)
			protected.GET("/ports", h.Port.ListPorts)
			protected.GET("/ports/:id", h.Port.GetPort)
			protected.GET("/vessels", h.Vessel.ListVessels)
			protected.GET("/vessels/:id", h.Vessel.GetVessel)
			protected.GET("/shipping-lines", h.ShippingLine.ListLines)
			protected.GET("/shipping-lines/:id", h.ShippingLine.GetLine)
			protected.GET("/shipping-lines/:id/port-sequence", h.ShippingLine.GetPortSequence)

			// 订单支付
			protected.POST("/orders/:id/pay", h.Order.PayOrder)

			// Excel 导入导出
			protected.GET("/export/ports", h.ImportExport.ExportPorts)
			protected.POST("/import/ports", h.ImportExport.ImportPorts)
			protected.GET("/export/vessels", h.ImportExport.ExportVessels)
			protected.POST("/import/vessels", h.ImportExport.ImportVessels)
			protected.GET("/export/shipping-lines", h.ImportExport.ExportShippingLines)
			protected.POST("/import/shipping-lines", h.ImportExport.ImportShippingLines)
			protected.GET("/export/orders", h.ImportExport.ExportOrders)

			// 通知（列表 + 已读标记）
			protected.GET("/notifications", h.Notification.ListNotifications)
			protected.PUT("/notifications/:id/read", h.Notification.MarkAsRead)

			// 报表统计
			protected.GET("/reports/orders", h.Report.OrderStatistics)
			protected.GET("/reports/voyage-utilization", h.Report.VoyageUtilization)

			// 航次管理（船公司查看自己的靠泊记录）
			protected.GET("/voyages/my", h.Berthing.ListByCompany)

			// 靠泊管理（更新实际到达/出发时间）
			protected.PUT("/berthings/:id/actual-times", h.Berthing.UpdateActualTimes)

			// 货主查看船公司列表
			protected.GET("/shipping-companies", h.ShippingCompany.List)

			// 管理员接口（仅限 admin 角色，额外叠加 RequireRole）
			adminGroup := protected.Group("/admin")
			adminGroup.Use(authMw.RequireRole("admin"))
			{
				adminGroup.GET("/list", h.Admin.List)
				adminGroup.GET("/cargo/list", h.Cargo.ListAllCargos)
				adminGroup.POST("/register", h.Admin.Create)
				adminGroup.POST("/password/:id", h.Admin.UpdatePassword)
				adminGroup.POST("/notifications", h.Notification.SendNotification)
				adminGroup.GET("/shipper/list", h.ShipperCompany.List)
				adminGroup.POST("/shipper/:id/update", h.ShipperCompany.UpdateByAdmin)
				adminGroup.POST("/shipper/:id/delete", h.ShipperCompany.Delete)
				adminGroup.GET("/shipping/list", h.ShippingCompany.List)
				adminGroup.POST("/shipping/:id/update", h.ShippingCompany.UpdateByAdmin)
				adminGroup.POST("/shipping/:id/delete", h.ShippingCompany.Delete)
				adminGroup.POST("/ports", h.Port.CreatePort)
				adminGroup.PUT("/ports/:id", h.Port.UpdatePort)
				adminGroup.DELETE("/ports/:id", h.Port.DeletePort)
				adminGroup.POST("/vessels", h.Vessel.CreateVessel)
				adminGroup.PUT("/vessels/:id", h.Vessel.UpdateVessel)
				adminGroup.DELETE("/vessels/:id", h.Vessel.DeleteVessel)
				adminGroup.POST("/shipping-lines", h.ShippingLine.CreateLine)
				adminGroup.PUT("/shipping-lines/:id", h.ShippingLine.UpdateLine)
				adminGroup.DELETE("/shipping-lines/:id", h.ShippingLine.DeleteLine)
				adminGroup.POST("/cities", h.City.CreateCity)
				adminGroup.PUT("/cities/:id", h.City.UpdateCity)
				adminGroup.DELETE("/cities/:id", h.City.DeleteCity)
			}
		}
	}

	// ── 404 兜底 ──
	// 所有未匹配的路由返回 404。
	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": 404, "message": "not found"})
	})
	return r
}
