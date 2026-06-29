================================================================================
                 航运管理系统后端 Service 模块功能说明
                 版本：1.0
                 生成日期：2026-06-05
================================================================================

本模块位于 internal/service 目录下，实现了所有核心业务逻辑，包括订单管理、
运力分配、推荐排序、账户管理、基础数据查询等。所有 service 均通过依赖注入
使用 DAO 层接口，支持事务、并发锁、错误码、批量操作等工业化特性。

================================================================================
1. common.go —— 通用工具函数
================================================================================

本文件提供其他 service 共用的基础设施，不包含具体业务逻辑。

【类型定义】
- ServiceError: 自定义错误类型，包含 Code 和 Message 字段，便于上层区分错误种类。
- 错误码常量: ErrCodeOrderNotFound, ErrCodeInsufficientCap, ErrCodeLockFailed,
  ErrCodeInvalidPortSeq, ErrCodeNoCargoNote, ErrCodeVesselNotFound,
  ErrCodeLineNotFound。

【函数列表】
- GenerateOrderNo() string
  生成唯一订单号，格式为 "ORD20250605" + 8位随机十六进制字符，用于订单创建。

- HashPassword(plain string) (string, error)
  使用 bcrypt 算法对明文密码进行哈希，用于用户注册和密码修改。

- CheckPasswordHash(plain, hash string) bool
  验证明文密码是否与哈希值匹配，用于登录。

- AcquireLock(tx *gorm.DB, lockName string, timeoutSec int) (bool, error)
  获取 MySQL 应用锁（GET_LOCK），用于并发控制。返回是否成功获取锁。

- ReleaseLock(tx *gorm.DB, lockName string) error
  释放 MySQL 应用锁。

- VoyageLockKey(lineID, vesselID int64, voyageDate string) string
  生成航次锁的唯一键名，例如 "voyage_123_456_2025-06-05"。

- CalculateSegments(portIDs []int64, startPortID, endPortID int64) ([][2]int64, error)
  根据港口 ID 序列（按顺序）和起运港、目的港，计算所有经过的相邻港口对（航段）。
  例如：portIDs = [1001,1005,1008], start=1001, end=1008 → 返回 [(1001,1005), (1005,1008)]。

- ParsePortSequence(seqStr string) ([]int64, error)
  将 JSON 字符串（如 "[1,2,3]"）解析为 []int64 切片，用于解析 shipping_line.port_sequence。

- MustParseDate(s string) time.Time
  将 "2006-01-02" 格式的字符串转换为 time.Time，忽略错误。

- PtrInt8(v int8) *int8
  返回 int8 值的指针，用于填充 model 中的指针字段。

================================================================================
2. order_service.go —— 订单核心业务
================================================================================

本文件处理订单的完整生命周期：创建、取消、状态更新、查询。

【数据结构】
- CreateOrderRequest: 创建订单的请求参数，包含货主ID、航线ID、船次ID、航次日期、
  起运港ID、目的港ID、货物列表等。
- CargoItem: 订单中的单个货物明细，包含名称、类型、重量、体积、单价、小计等。

【接口 OrderService】
- CreateOrder(ctx, req) (*model.ShippingOrder, error)
  核心方法，执行以下步骤：
    1. 校验货物列表非空，计算总重量、总体积、总费用。
    2. 获取船舶最大载重吨位。
    3. 解析航线港口序列，得到订单需要经过的所有航段。
    4. 根据起运港和目的港，从 voyage_cargo_note 表中查找对应的装货单（LOAD）和卸货单（UNLOAD）。
    5. 开启数据库事务，获取航次应用锁（防止并发超卖）。
    6. 对每个航段锁定已有占位记录（SELECT FOR UPDATE），并计算剩余容量。
    7. 若任何航段容量不足，返回业务错误。
    8. 生成订单号，插入 shipping_order 记录。
    9. 批量插入 order_cargo 记录（每个货物一条）。
    10. 批量插入 segment_capacity_usage 记录（每个航段一条，占用吨位等于订单总重量）。
    11. 更新装货单和卸货单的 cumulative_booked_capacity_ton（累加当前订单重量）。
    12. 提交事务，返回订单对象。
  支持一个订单多种货物、多航段占用、并发安全。

- CancelOrder(ctx, orderID) error
  取消订单，业务逻辑：
    1. 锁定并查询订单，同时预加载 LoadNote 和 UnloadNote。
    2. 若订单已取消，返回错误。
    3. 获取航次锁（基于装货单中的航次信息）。
    4. 在装货单和卸货单上减去该订单占用的吨位（更新累计吨位）。
    5. 软删除 shipping_order（设置 delete_time）。
    6. 软删除所有关联的 order_cargo。
    7. 物理删除 segment_capacity_usage 记录（释放运力）。
    8. 提交事务。

- UpdateOrderStatus(ctx, orderID, newStatus) error
  简单更新订单的状态字段（如 1-已确认, 2-运输中, 3-已完成, 4-已取消）。

- GetOrderByID(ctx, orderID) (*model.ShippingOrder, error)
  根据订单 ID 查询订单，自动预加载关联的货物（OrderCargos）、装货单、卸货单。

- ListOrdersByShipper(ctx, shipperCompanyID, page, pageSize, sortBy, sortOrder) ([]model.ShippingOrder, int64, error)
  分页查询某个货主的所有订单，支持动态排序（sortBy 可选值：create_time, order_no, total_weight_ton, order_status；sortOrder 为 asc 或 desc）。默认按 create_time desc。

【依赖的 DAO】: ShippingOrderDAO, OrderCargoDAO, SegmentCapacityUsageDAO,
  VoyageCargoNoteDAO, VesselDAO, ShippingLineDAO。

================================================================================
3. voyage_service.go —— 航次与运力推荐
================================================================================

本文件提供航次剩余容量查询和可用航次推荐（按剩余容量降序排序）。

【数据结构】
- VoyageRecommendation: 推荐结果，包含航线ID、船次ID、航次日期、船名、航线名、剩余容量。

【接口 VoyageService】
- GetRemainingCapacity(ctx, lineID, vesselID, voyageDate, startPortID, endPortID) (float64, error)
  查询指定航次、指定航段的当前剩余载重吨位。计算方式：船舶最大载重 - 该航段已占用吨位（从 segment_capacity_usage 聚合）。用于下单前的容量检查。

- RecommendVoyages(ctx, startPortID, endPortID, requiredTon) ([]VoyageRecommendation, error)
  核心推荐算法：
    1. 查询所有未删除的航线（shipping_line）。
    2. 对每条航线，解析其 port_sequence，检查起运港是否在目的港之前。
    3. 找出该航线下的所有航次（从 voyage_cargo_note 获取不同的 vessel_id, voyage_date）。
    4. 对每个航次，计算从起运港到目的港所有航段的最小剩余容量。
    5. 如果最小剩余容量 >= 需求吨位，则视为可用航次。
    6. 将所有可用航次按剩余容量降序排序（最宽松的先推荐）。
  返回推荐列表，供用户选择后下单。

【依赖的 DAO】: ShippingLineDAO, VesselDAO, VoyageCargoNoteDAO, SegmentCapacityUsageDAO。

================================================================================
4. company_service.go —— 货主与航运公司账户服务
================================================================================

本文件处理 shipper_company 和 shipping_company 两种角色的账户管理。两个 service
逻辑完全相同，故合并说明。

【接口 ShipperCompanyService】（ShippingCompanyService 类似）
- Register(company *model.ShipperCompany, plainPassword string) error
  注册新公司：对明文密码进行 bcrypt 哈希，然后调用 DAO 的 Create 方法插入数据库。

- Login(username, plainPassword string) (*model.ShipperCompany, error)
  登录验证：根据用户名查询公司，校验密码哈希，检查 account_status 是否为 1（启用），
  返回公司对象。密码错误或账户禁用均返回 "invalid username or password" 业务错误。

- UpdatePassword(companyID int64, oldPassword, newPassword string) error
  修改密码：先根据 ID 查询公司，验证旧密码，对新密码哈希后更新数据库。

【依赖的 DAO】: ShipperCompanyDAO / ShippingCompanyDAO。

================================================================================
5. admin_service.go —— 管理员账户服务
================================================================================

处理 admin 表的账户管理，功能与公司服务类似。

【接口 AdminService】
- Create(admin *model.Admin, plainPassword string) error
  创建管理员：密码哈希后存储。

- Login(username, plainPassword string) (*model.Admin, error)
  管理员登录：验证用户名和密码哈希。

- UpdatePassword(adminID int64, oldPassword, newPassword string) error
  修改管理员密码：验证旧密码后更新。

【依赖的 DAO】: AdminDAO。

================================================================================
6. port_service.go —— 港口基础查询
================================================================================

简单的数据透传服务，封装 PortDAO，用于上层（如 handler）调用。

【接口 PortService】
- GetPortByID(id int64) (*model.Port, error)
  根据 ID 查询单个港口（未删除的）。

- ListPorts(page, pageSize int) ([]model.Port, int64, error)
  分页查询所有未删除的港口。

- ListPortsByCity(cityID int64, page, pageSize int) ([]model.Port, int64, error)
  分页查询某个城市下的所有港口。

【依赖的 DAO】: PortDAO。

================================================================================
7. vessel_service.go —— 船舶基础查询
================================================================================

封装 VesselDAO，提供船舶查询。

【接口 VesselService】
- GetVesselByID(id int64) (*model.Vessel, error)
  根据 ID 查询单艘船舶（未删除的）。

- ListVessels(page, pageSize int) ([]model.Vessel, int64, error)
  分页查询所有未删除的船舶。

- ListVesselsByCompany(companyID int64, page, pageSize int) ([]model.Vessel, int64, error)
  分页查询某个航运公司下的所有船舶。

【依赖的 DAO】: VesselDAO。

================================================================================
8. shipping_line_service.go —— 航线基础查询与端口序列解析
================================================================================

封装 ShippingLineDAO，并提供解析 port_sequence 的方法。

【接口 ShippingLineService】
- GetLineByID(id int64) (*model.ShippingLine, error)
  根据 ID 查询单条航线（未删除的）。

- ListLines(page, pageSize int) ([]model.ShippingLine, int64, error)
  分页查询所有未删除的航线。

- ListLinesByCompany(companyID int64, page, pageSize int) ([]model.ShippingLine, int64, error)
  分页查询某个航运公司下的所有航线。

- GetPortSequence(lineID int64) ([]int64, error)
  获取指定航线的 port_sequence 字段，并将其从 JSON 字符串解析为 []int64 切片，
  供其他 service（如 order_service、voyage_service）使用。

【依赖的 DAO】: ShippingLineDAO。

================================================================================
9. 模块间协作典型流程
================================================================================

以下是一个完整的下单流程，展示了各 service 如何协作：

1. 用户（货主）通过前端输入起运港、目的港、货物重量等，请求推荐航次。
2. Handler 调用 VoyageService.RecommendVoyages() 获得按剩余容量排序的可用航次列表。
3. 用户选择一个航次，填写货物明细，提交订单。
4. Handler 调用 OrderService.CreateOrder()，传入参数。
5. OrderService 内部：
   - 调用 ShippingLineService.GetLineByID() 获取航线信息。
   - 调用 ShippingLineService.GetPortSequence() 解析港口序列。
   - 调用 VesselService.GetVesselByID() 获取船舶最大载重。
   - 调用 VoyageCargoNoteDAO.FindByPortAndOp() 获取装/卸货单。
   - 调用 SegmentCapacityUsageDAO.GetOccupiedTons() 检查每个航段容量。
   - 在事务中插入订单、货物、航段占用，并更新累计吨位。
6. 返回成功，订单创建完成。

其他独立功能如注册、登录、基础数据查询等，可直接调用相应的 service 方法。

================================================================================
10. 错误处理与日志说明
================================================================================

- 所有 service 方法在遇到业务错误（如运力不足、订单不存在）时，返回 ServiceError
  类型，其中包含 Code 和 Message。上层（如 handler）可以根据 Code 转换为 HTTP 状态码。
- 系统级错误（如数据库连接失败）直接返回原始 error。
- 关键操作（创建订单、取消订单）已包含事务和锁，失败会自动回滚。
- 建议在 handler 层记录每个 service 调用的耗时和错误日志。

================================================================================
2026-06-11 更新内容
================================================================================

1. 订单服务增强
   - 新增 `GetOrderTracking` 方法及 `OrderTracking` 结构体，实现货物跟踪功能，返回订单的装卸货时间、靠泊计划/实际时间、船舶/航线名称等详细信息。
   - 在 `CreateOrder` 中集成运费自动计算：基于航线距离、货物总重量和配置费率，覆盖前端传入的 `total_cost`。
   - 在 `UpdateOrderStatus` 中集成 WebSocket 推送：订单状态变更后，调用 `WebSocketService.PushOrderStatusUpdate` 向对应货主实时推送消息。

2. 新增服务
   - `NotificationService`：提供消息通知的发送、查询（分页）、标记已读功能。通知存储于内存（可扩展为数据库或 Redis）。
   - `ReportService`：提供报表统计功能，包括订单汇总（按日期范围统计总订单数、总重量/体积/运费、各状态数量）和航次载重利用率（已占吨位/船舶最大载重百分比）。

3. 缓存集成
   - `RecommendVoyages` 方法结果缓存 1 分钟（TTL），订单创建或取消时自动清除相关缓存键，保证数据一致性。

4. 依赖调整
   - 订单服务构造函数 `NewOrderService` 新增 `WebSocketService` 参数，用于推送通知。
   - 错误处理统一迁移至 `pkg/errors`，不再使用 `ServiceError`。
   - 密码哈希函数迁移至 `pkg/crypto`，订单号生成、航段计算等移至 `biz` 层。

5. 性能与可靠性
   - 航次推荐结果缓存降低了数据库压力。
   - 订单创建和取消时使用 `cache.DeletePrefix` 高效清除缓存前缀。


================================================================================
                              文档结束
================================================================================