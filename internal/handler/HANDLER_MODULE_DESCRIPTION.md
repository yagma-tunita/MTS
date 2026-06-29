================================================================================
                INTERNAL/HANDLER 业务控制器模块说明
                版本：1.0
                生成日期：2026-06-09
================================================================================

本模块位于 backend/internal/handler 目录，负责处理 HTTP 请求，调用 service 层
完成业务逻辑，并使用 pkg/response 返回统一格式的响应。handler 层与 net 层分离，
保持 net 层仅做基础设施。

--------------------------------------------------------------------------------
1. 目录结构
--------------------------------------------------------------------------------
internal/handler/
├── handler.go          # 统一依赖注入容器，聚合所有 handler
├── auth.go             # 认证控制器（登录、刷新令牌）
├── order.go            # 订单控制器（创建、取消、状态更新、列表）
├── voyage.go           # 航次控制器（推荐航次）
├── company.go          # 货主/航运公司控制器（注册、改密）
├── admin.go            # 管理员控制器（创建管理员、改密）
├── port.go             # 港口控制器（查询）
├── vessel.go           # 船舶控制器（查询）
└── shipping_line.go    # 航线控制器（查询、获取港口序列）

--------------------------------------------------------------------------------
2. 控制器功能列表
--------------------------------------------------------------------------------
2.1 auth.go
    POST   /api/v1/auth/login          # 登录，返回 access_token 和 refresh_token
    POST   /api/v1/auth/refresh        # 刷新访问令牌

2.2 order.go
    POST   /api/v1/orders              # 创建订单（支持多货物、多航段）
    GET    /api/v1/orders/:id          # 获取订单详情
    POST   /api/v1/orders/:id/cancel   # 取消订单
    PUT    /api/v1/orders/:id/status   # 更新订单状态
    GET    /api/v1/orders              # 按货主分页查询订单（支持排序）

2.3 voyage.go
    GET    /api/v1/voyages/recommend   # 推荐可用航次（按剩余容量降序）

2.4 company.go
    POST   /api/v1/shipper/register    # 货主公司注册
    POST   /api/v1/shipper/password    # 货主修改密码
    POST   /api/v1/shipping/register   # 航运公司注册
    POST   /api/v1/shipping/password   # 航运公司修改密码

2.5 admin.go
    POST   /api/v1/admin/register      # 创建管理员
    POST   /api/v1/admin/password      # 管理员修改密码

2.6 port.go
    GET    /api/v1/ports               # 港口列表（分页）
    GET    /api/v1/ports/:id           # 港口详情
    GET    /api/v1/ports?city_id=...   # 按城市查询港口

2.7 vessel.go
    GET    /api/v1/vessels             # 船舶列表（分页）
    GET    /api/v1/vessels/:id         # 船舶详情
    GET    /api/v1/vessels?shipping_company_id=... # 按船公司查询

2.8 shipping_line.go
    GET    /api/v1/shipping-lines               # 航线列表（分页）
    GET    /api/v1/shipping-lines/:id           # 航线详情
    GET    /api/v1/shipping-lines/:id/port-sequence # 获取港口序列（JSON 数组）

--------------------------------------------------------------------------------
3. 依赖注入
--------------------------------------------------------------------------------
handler.go 中的 NewHandlers 函数接收所有 service 和 jwt 服务，构建 Handlers 结构体。
该结构体在 main.go 中创建后传递给 router.Setup。

示例：
    handlers := handler.NewHandlers(orderSvc, voyageSvc, ..., jwtSvc)
    r := router.Setup(handlers, jwtSvc)

--------------------------------------------------------------------------------
4. 请求与响应格式
--------------------------------------------------------------------------------
- 请求体使用 JSON，通过 pkg/validator 进行参数校验。
- 响应统一使用 pkg/response 包：
    Success       -> {"code":0, "message":"success", "data": {...}}
    SuccessPage   -> 增加 meta 字段（page, page_size, total, total_pages）
    ErrorWithCode -> 根据错误码自动映射 HTTP 状态码和业务码

--------------------------------------------------------------------------------
5. 错误处理
--------------------------------------------------------------------------------
- 业务错误（如参数非法、资源不存在）返回 AppError，包含错误码（如 1001, 1003）。
- 系统错误（如数据库连接失败）返回 500 Internal Server Error。
- 认证失败返回 401 Unauthorized。
- 权限不足返回 403 Forbidden。

--------------------------------------------------------------------------------
6. 与 service 层的协作
--------------------------------------------------------------------------------
- handler 不直接操作 DAO，不包含事务逻辑，所有业务逻辑委托给 service。
- handler 负责：
    - 解析请求参数
    - 校验参数
    - 调用 service 方法
    - 将 service 返回结果封装为 HTTP 响应
- handler 不处理数据库事务、锁、缓存等，这些由 service 和 biz 完成。

--------------------------------------------------------------------------------
7. 注意事项
--------------------------------------------------------------------------------
- 所有 handler 方法都已实现，但部分依赖的 service 方法（如 ListOrdersByShipper 中的排序参数）需要确保 service 层支持。
- 密码修改、注册等接口已经在 service 层实现了 bcrypt 哈希。
- 文件上传、CSV 导入等高级功能未包含在当前版本，可根据需求扩展。
- 所有 handler 均使用 context 传递请求上下文，支持超时控制。

================================================================================
2026-06-11 更新内容
================================================================================

1. 新增订单货物跟踪接口
   - 路径：GET /api/v1/orders/:id/tracking
   - 功能：返回订单的装卸货时间、起运/目的港靠泊计划/实际时间、船舶名称、航线名称等详细跟踪信息。

2. 新增 Excel 批量导入导出模块（ImportExportHandler）
   - 导出接口（GET）：
     * /api/v1/export/ports          - 导出港口列表为 Excel
     * /api/v1/export/vessels        - 导出船舶列表为 Excel
     * /api/v1/export/shipping-lines - 导出航线列表为 Excel
     * /api/v1/export/orders         - 导出订单列表为 Excel（需指定 shipper_company_id）
   - 导入接口（POST）：
     * /api/v1/import/ports          - 从 Excel 导入港口
     * /api/v1/import/vessels        - 从 Excel 导入船舶
     * /api/v1/import/shipping-lines - 从 Excel 导入航线
   - 所有导入导出接口均需 JWT 认证，管理员可操作全部，货主仅可导出自己的订单。

3. 新增消息通知模块（NotificationHandler）
   - GET    /api/v1/notifications               - 获取当前用户的通知列表（分页）
   - PUT    /api/v1/notifications/:id/read      - 标记指定通知为已读
   - POST   /api/v1/admin/notifications         - 管理员发送通知（需 admin 角色）
   - 通知存储于内存（可扩展为数据库或 Redis），支持类型、标题、内容、自定义数据字段。

4. 新增报表模块（ReportHandler）
   - GET    /api/v1/reports/orders              - 订单统计报表：按日期范围统计总订单数、总重量/体积/运费、各状态数量
   - GET    /api/v1/reports/voyage-utilization  - 航次载重利用率报表：返回某航次的最大载重、已占吨位、利用率百分比

5. 依赖注入更新
   - NewHandlers 函数新增参数：`importExportSvc`, `notifSvc`, `reportSvc`
   - 对应的 Handlers 结构体增加了 `ImportExport`、`Notification`、`Report` 字段。

6. 控制器文件新增
   - import_export.go：实现所有导入导出接口
   - notification.go：实现通知相关接口
   - report.go：实现报表相关接口

7. 订单控制器增强
   - order.go 中新增 `GetOrderTracking` 方法。


================================================================================
                              文档结束
================================================================================