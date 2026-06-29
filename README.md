================================================================================
        航运物流管理系统 (MTS) 后端 - 完整项目文档
        版本：1.1
        生成日期：2026-06-11
================================================================================

1. 项目简介
--------------------------------------------------------------------------------
MTS (Maritime Transport System) 是一个工业化的航运物流管理平台后端，提供订单管理、
航次推荐、货物跟踪、消息通知、报表统计、Excel导入导出及实时WebSocket推送等功能。
系统基于Go语言开发，采用分层架构（model/dao/biz/service/handler），确保高内聚低耦合，
适用于中小型航运企业或教学演示。

2. 技术栈
--------------------------------------------------------------------------------
- Go 1.21+
- Web框架：Gin
- ORM：GORM (MySQL)
- 日志：slog + lumberjack（文件轮转）
- 认证：JWT (HMAC-SHA256)
- 缓存：内存缓存 (go-cache)
- Excel处理：excelize
- WebSocket：gorilla/websocket
- 配置：YAML + 环境变量
- API文档：Swagger (swaggo)
- 可观测性：pprof、慢查询日志

3. 项目结构
--------------------------------------------------------------------------------
backend/                         # 项目根目录
├── app/                         # 应用入口（唯一的main包）
│   └── main.go                  # 主程序，负责配置加载、依赖注入、启动HTTP服务
├── internal/                    # 私有业务代码（不可被外部导入）
│   ├── model/                   # GORM实体，定义表结构、软删除、关联关系
│   ├── dao/                     # 数据访问层，提供基础CRUD及自定义查询（如FindByPortAndOp）
│   ├── biz/                     # 领域逻辑层：运力校验、航段计算、状态机、推荐排序等
│   ├── service/                 # 应用服务层：事务管理、业务编排、调用biz和dao
│   └── handler/                 # HTTP控制器层：解析请求、调用service、统一响应
├── net/                         # HTTP基础设施（与业务解耦）
│   ├── middleware/              # 全局中间件：认证、CORS、日志、限流、恢复
│   ├── protect/                 # 安全防御：安全头、蜜罐、IP黑名单、请求守卫
│   ├── websocket/               # WebSocket服务：连接管理、心跳保活、定向推送
│   └── router/                  # 路由注册：组装中间件、绑定handler
├── pkg/                         # 公共基础设施（可被其他项目导入）
│   ├── config/                  # 配置加载（YAML + 环境变量）
│   ├── logger/                  # 日志初始化（slog + lumberjack）
│   ├── database/                # MySQL连接池、健康检查
│   ├── cache/                   # 内存缓存（TTL、前缀删除）
│   ├── jwt/                     # JWT生成、解析、刷新
│   ├── crypto/                  # bcrypt哈希、AES、MD5、随机字符串
│   ├── errors/                  # 统一错误码与AppError
│   ├── response/                # 统一JSON响应（成功、分页、错误）
│   ├── validator/               # 参数校验（基于go-playground/validator）
│   ├── excel/                   # Excel读写（基于excelize）
│   ├── timeutil/                # 东八区时间、日期解析/格式化
│   ├── idgen/                   # 雪花算法ID生成器
│   └── fileutil/                # 文件操作（原子写、目录拷贝、文件锁）
├── sql/                         # 数据库脚本
│   └── tables_mysql.sql         # 建表语句（含软删除、索引）
├── docs/                        # Swagger生成的文档（自动生成，可忽略）
├── config.yaml                  # 配置文件示例（实际使用时放在根目录）
├── go.mod
├── go.sum
└── README.txt                   # 本文件

4. 核心功能详解
--------------------------------------------------------------------------------
4.1 订单管理
    - 创建订单：支持多货物、多航段，自动计算运费（基于航线距离+费率），校验每个航段剩余运力，
      使用MySQL应用锁（GET_LOCK）防止并发超卖，事务中批量插入订单货物和航段占用记录，
      同时更新装卸货单的累计已订吨位。
    - 取消订单：软删除订单及货物，物理删除航段占用记录，恢复累计吨位。
    - 更新订单状态：基于状态机（草稿→已确认→运输中→已完成/取消），成功后WebSocket推送给货主。
    - 查询订单：支持单个查询（预加载货物、装卸货单）、分页查询（动态排序）。
    - 货物跟踪：返回订单的装卸货时间、起运/目的港靠泊计划/实际时间、船舶名称、航线名称。

4.2 航次推荐
    - 根据起运港、目的港、需求吨位，筛选所有可行航次。
    - 计算每个航次从起运港到目的港所经过的所有航段，取最小剩余容量作为瓶颈。
    - 过滤瓶颈容量 >= 需求吨位的航次，按剩余容量降序排序，结果缓存1分钟。

4.3 用户与权限
    - 角色：货主(shipper)、航运公司(shipping)、管理员(admin)。
    - 公开注册：货主和航运公司可自行注册；管理员只能由已有管理员创建。
    - JWT认证：登录时返回access_token（15分钟）和refresh_token（7天）。
    - 授权：路由级中间件检查role，管理员接口需`role=admin`。

4.4 基础数据管理
    - 城市、港口、泊位、船舶、航线、航次靠泊、装卸货单的CRUD操作。
    - 支持通过Excel批量导入导出港口、船舶、航线、订单（按货主）。

4.5 运费自动计算
    - 在创建订单时，根据货物总重量、航线总海里数、配置的基础费率及货物类型系数，
      计算订单总费用，并覆盖前端传入的total_cost值，确保数据一致性。

4.6 Excel导入导出
    - 导出：GET /export/ports, /export/vessels, /export/shipping-lines, /export/orders
    - 导入：POST /import/ports, /import/vessels, /import/shipping-lines
    - 支持字段：ID、名称、代码、关联ID等，使用excelize库实现。

4.7 消息通知
    - 管理员可向指定用户发送通知（标题、内容、类型、自定义数据）。
    - 用户可查询自己的未读/已读通知列表（分页），并标记已读。
    - 当前通知存储于内存，生产环境可替换为数据库或Redis。

4.8 报表统计
    - 订单统计：按日期范围统计总订单数、总重量、总体积、总运费，以及完成/取消/运输中数量。
    - 航次载重利用率：查询某航次当前已占吨位，除以船舶最大载重，返回百分比。

4.9 实时WebSocket推送
    - 端点：ws://host/ws?token=<access_token>
    - 连接时验证JWT，建立连接后自动发送心跳（ping/pong）。
    - 订单状态变更时，服务端主动向对应货主推送JSON消息：
      {"type":"order_status_update","order_id":4,"status":2,"timestamp":...}

5. 环境要求
--------------------------------------------------------------------------------
- Go 1.21 或更高版本
- MySQL 5.7+ 或 8.0+
- （可选）Node.js 及 npm（仅用于测试WebSocket的wscat工具）

6. 配置说明
--------------------------------------------------------------------------------
配置文件为根目录下的 config.yaml，支持环境变量覆盖。

示例配置文件内容：
--------------------------------------------------------------------------------
server:
  port: "8080"                     # HTTP监听端口
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s

database:
  driver: mysql
  dsn: "root:password@tcp(127.0.0.1:3306)/mts?charset=utf8mb4&parseTime=True&loc=Local"
  max_open_conns: 25
  max_idle_conns: 10
  conn_max_lifetime: 5m
  conn_max_idle_time: 5m

log:
  level: info                      # debug/info/warn/error
  format: json                     # json 或 text
  output_path: "logs/app.log"      # 或 stdout
  max_size: 100                    # MB
  max_backups: 10
  max_age: 30                      # 天
  compress: true

jwt:
  secret: "your-production-secret-key"   # 必须修改，不可使用默认值
  access_expire: 15m
  refresh_expire: 168h

freight:
  base_rate_per_ton_nm: 0.05       # 元/吨/海里
  cargo_type_factors:
    bulk: 1.0
    container: 1.2
    liquid: 1.1
--------------------------------------------------------------------------------

环境变量覆盖示例：
  set DB_DSN=root:newpass@tcp(127.0.0.1:3306)/mts?...
  set JWT_SECRET=mysecret

7. 启动步骤
--------------------------------------------------------------------------------
7.1 安装依赖
    cd backend
    go mod tidy

7.2 创建数据库并执行建表脚本
    mysql -u root -p < sql/tables_mysql.sql

7.3 修改配置文件
    复制 config.yaml 到项目根目录，并按实际情况修改数据库连接、JWT密钥等。

7.4 生成Swagger文档（可选）
    go install github.com/swaggo/swag/cmd/swag@latest
    swag init -g app/main.go -o docs

7.5 启动服务
    go run ./app

    观察日志输出，应看到：
    server started port=8080

7.6 验证健康检查
    访问 http://localhost:8080/health
    返回 {"status":"ok"}

8. API接口概览
--------------------------------------------------------------------------------
详细接口说明请参考 FOR_FRONTEND.txt 或 Swagger UI (http://localhost:8080/swagger/index.html)。

主要端点：
- 认证：POST /api/v1/auth/login, POST /api/v1/auth/refresh
- 订单：POST /api/v1/orders, GET /api/v1/orders/:id, POST /api/v1/orders/:id/cancel,
        PUT /api/v1/orders/:id/status, GET /api/v1/orders, GET /api/v1/orders/:id/tracking
- 航次推荐：GET /api/v1/voyages/recommend
- 港口：GET /api/v1/ports, GET /api/v1/ports/:id
- 船舶：GET /api/v1/vessels, GET /api/v1/vessels/:id
- 航线：GET /api/v1/shipping-lines, GET /api/v1/shipping-lines/:id,
        GET /api/v1/shipping-lines/:id/port-sequence
- 导入导出：GET /export/ports, POST /import/ports, GET /export/vessels, POST /import/vessels,
           GET /export/shipping-lines, POST /import/shipping-lines, GET /export/orders
- 通知：GET /api/v1/notifications, PUT /api/v1/notifications/:id/read,
        POST /api/v1/admin/notifications (需admin)
- 报表：GET /api/v1/reports/orders, GET /api/v1/reports/voyage-utilization
- WebSocket：ws://localhost:8080/ws?token=<access_token>

9. 测试示例（使用PowerShell）
--------------------------------------------------------------------------------
9.1 健康检查
    Invoke-RestMethod -Uri http://localhost:8080/health

9.2 注册货主
    $body = @{company_name="测试";login_username="test1";password="123456"} | ConvertTo-Json
    Invoke-RestMethod -Uri http://localhost:8080/api/v1/shipper/register -Method POST -Body $body -ContentType "application/json"

9.3 登录获取Token
    $login = @{username="test1";password="123456";role="shipper"} | ConvertTo-Json
    $resp = Invoke-RestMethod -Uri http://localhost:8080/api/v1/auth/login -Method POST -Body $login -ContentType "application/json"
    $token = $resp.data.access_token

9.4 创建订单（需数据库中有航线、船舶、港口等基础数据）
    $headers = @{Authorization="Bearer $token"}
    $order = @{
        shipper_company_id=1
        city_id=1
        line_id=1
        vessel_id=1
        voyage_date=(Get-Date -Format "yyyy-MM-dd")
        start_port_id=1
        end_port_id=2
        cargo_items=@(@{cargo_name="Coal";cargo_type="bulk";quantity=100;weight_ton=500;volume_cub_m=200;unit_price=10.5;subtotal=5250})
    } | ConvertTo-Json -Depth 3
    Invoke-RestMethod -Uri http://localhost:8080/api/v1/orders -Method POST -Headers $headers -Body $order -ContentType "application/json"

9.5 更新订单状态（触发WebSocket推送）
    $status = @{status=2} | ConvertTo-Json
    Invoke-RestMethod -Uri "http://localhost:8080/api/v1/orders/1/status" -Method PUT -Headers $headers -Body $status -ContentType "application/json"

9.6 连接WebSocket（需安装wscat）
    npm install -g wscat
    wscat -c "ws://localhost:8080/ws?token=$token"
    （另开终端执行上一步状态更新，观察wscat窗口收到推送消息）

10. 常见问题及解决方法
--------------------------------------------------------------------------------
Q1: 启动时提示 "JWT secret must be set (not default)"
A1: 修改 config.yaml 中 jwt.secret 的值，不能使用默认字符串 "change-me-in-production"。

Q2: WebSocket 连接返回 401
A2: 检查 token 是否有效（可通过调用 /api/v1/ports 验证），且 WebSocket 路径为 /ws（不是 /api/v1/ws）。

Q3: 订单创建失败，错误 "no LOAD cargo note for start port"
A3: 需要在数据库中为指定的航线、船舶、日期预创建航次靠泊记录和对应的装卸货单（operation_type='LOAD'/'UNLOAD'）。
    可执行 sql/insert_test_data.sql 中的示例数据。

Q4: 订单总运费为0
A4: 检查 shipping_line 表中 total_distance_nm 字段是否为 NULL 或 0；检查 config.yaml 中 freight.base_rate_per_ton_nm 是否设置。

Q5: Excel 导入失败，提示 "invalid format"
A5: 确保 Excel 文件第一列为表头，列顺序与导出的模板完全一致，且数据类型正确（例如 ID 列为数字）。

Q6: 限流返回 429
A6: 默认限流为每秒100个令牌，突发20。生产环境可根据需要调整 net/middleware/rate_limit.go 中的 DefaultRateLimiterConfig。

11. 性能与安全建议
--------------------------------------------------------------------------------
- 生产环境务必设置 gin.SetMode(gin.ReleaseMode)，关闭调试输出。
- 修改 config.yaml 中 JWT_SECRET 为强密码（至少32字节随机字符串）。
- 日志输出路径建议为文件（如 logs/app.log）并配置轮转。
- 如果多实例部署，需将内存缓存（pkg/cache）和通知存储（service/notification_service.go）
  替换为 Redis，同时限流中间件也需改用 Redis 实现分布式限流。
- 定期执行数据库备份，可增加定时任务清理超过30天的软删除记录。
- 启用 pprof 性能分析端点：设置环境变量 ENABLE_PPROF=true 后重启服务，访问 /debug/pprof。

================================================================================
                            文档结束
================================================================================