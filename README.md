# MTS（Maritime Transport System）航运物流管理系统

版本：1.2
生成日期：2026-07-02

---

## 目录

1. [项目简介](#1-项目简介)
2. [需求分析](#2-需求分析)
3. [数据库设计](#3-数据库设计)
4. [技术栈](#4-技术栈)
5. [项目结构](#5-项目结构)
6. [模块架构](#6-模块架构)
7. [核心功能详解](#7-核心功能详解)
8. [请求完整链路示例（创建订单）](#8-请求完整链路示例创建订单)
9. [API 接口总览](#9-api-接口总览)
10. [环境要求与启动](#10-环境要求与启动)
11. [配置说明](#11-配置说明)
12. [测试示例](#12-测试示例)
13. [常见问题](#13-常见问题)
14. [性能与安全建议](#14-性能与安全建议)

---

## 1. 项目简介

MTS 是一个工业化的航运物流管理平台后端，连接**货主（shipper）**和**船公司（shipping）**，提供货运订单的全生命周期管理。

### 核心业务流程

```
货主注册 → 查询航线/航次 → 选择航次下订单
    → 系统自动计算运费 + 校验船舶剩余运力
    → 订单确认 → 船公司安排运输 → 状态跟踪 → 完成
```

### 三种用户角色

| 角色 | 能力 | 注册方式 |
|------|------|---------|
| **shipper**（货主） | 下订单、查订单、查港口/船舶/航线/航次推荐 | 公开注册 |
| **shipping**（船公司） | 管理船舶、航线、航次数据 | 公开注册 |
| **admin**（管理员） | 管理所有数据、发通知、查报表 | 管理员邀请 |

---

## 2. 需求分析

### 2.1 功能需求

#### 公开功能（无需登录）

| 功能 | 端点 | 说明 |
|------|------|------|
| 健康检查 | `GET /health` | 服务存活检测 |
| 货主注册 | `POST /api/v1/shipper/register` | 创建货主账号，bcrypt 存密码 |
| 船公司注册 | `POST /api/v1/shipping/register` | 创建船公司账号 |
| 登录 | `POST /api/v1/auth/login` | 三种角色登录，返回 JWT 双令牌 |
| 刷新令牌 | `POST /api/v1/auth/refresh` | access_token 续期 |

#### 认证功能（需 JWT）

| 模块 | 功能 | 端点 |
|------|------|------|
| **订单管理** | 创建订单 | `POST /api/v1/orders` |
| | 订单详情 | `GET /api/v1/orders/:id` |
| | 取消订单 | `POST /api/v1/orders/:id/cancel` |
| | 更新状态 | `PUT /api/v1/orders/:id/status` |
| | 订单列表 | `GET /api/v1/orders` |
| | 货物跟踪 | `GET /api/v1/orders/:id/tracking` |
| **密码修改** | 货主 | `POST /api/v1/shipper/password/:id` |
| | 船公司 | `POST /api/v1/shipping/password/:id` |
| **数据查询** | 港口列表/详情 | `GET /api/v1/ports` / `GET /api/v1/ports/:id` |
| | 船舶列表/详情 | `GET /api/v1/vessels` / `GET /api/v1/vessels/:id` |
| | 航线列表/详情 | `GET /api/v1/shipping-lines` / `GET /api/v1/shipping-lines/:id` |
| | 航线港口序列 | `GET /api/v1/shipping-lines/:id/port-sequence` |
| **航次推荐** | 推荐航次 | `GET /api/v1/voyages/recommend` |
| **导入导出** | 导出港口/船舶/航线/订单 | `GET /api/v1/export/*` |
| | 导入港口/船舶/航线 | `POST /api/v1/import/*` |
| **通知** | 通知列表 | `GET /api/v1/notifications` |
| | 标记已读 | `PUT /api/v1/notifications/:id/read` |
| **报表** | 订单统计 | `GET /api/v1/reports/orders` |
| | 航次利用率 | `GET /api/v1/reports/voyage-utilization` |

#### 管理员功能（需 role=admin）

| 功能 | 端点 |
|------|------|
| 创建管理员 | `POST /api/v1/admin/register` |
| 修改密码 | `POST /api/v1/admin/password/:id` |
| 发送通知 | `POST /api/v1/admin/notifications` |

#### WebSocket 实时推送

| 功能 | 路径 |
|------|------|
| 订单状态推送 | `ws://host/ws?token=<access_token>` |

### 2.2 业务规则

```
1. 运费公式
   总运费 = 总重量(t) × 总海里(nm) × 基础费率(0.05) × 货类系数
   货类系数：bulk(散货)=1.0, container(集装箱)=1.2, liquid(液体)=1.1

2. 订单状态机
   0(草稿) → 1(已确认) → 2(运输中) → 3(已完成)
         ↘ 4(已取消) ←───────────────↙
   任何状态都可以取消

3. 运力校验
   订单占用航段 = 起运港到目的港之间的所有相邻港口对
   每个航段：max_deadweight - 已占吨位 - 新订单吨位 ≥ 0
   任一航段不足则拒绝订单

4. 并发控制
   使用 MySQL GET_LOCK("voyage_{line}_{vessel}_{date}") + SELECT FOR UPDATE

5. 软删除
   所有业务表通过 delete_time 标记删除，查询过滤 WHERE delete_time IS NULL
```

---

## 3. 数据库设计

### 3.1 13张表关系图

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│   city   │1──N│   port   │1──N│   berth  │
│ (城市)   │     │ (港口)   │     │ (泊位)   │
└──────────┘     └──────────┘     └──────────┘
                       │
                  N────┴────N
                   │         │
                   │    ┌────┴──────────────────┐
                   │    │    voyage_berthing     │ ←── shipping_line ──┐
                   │    │    (航次靠泊)           │     (航线)          │
                   │    └────────────────────────┘                    │
                   │                                                  │
                   │    ┌────────────────────────┐                    │
                   │    │   voyage_cargo_note     │ ←──────────────────┘
                   │    │   (航次货单)             │                    │
                   │    └────────────────────────┘               ┌─────┴──────────┐
                   │                                              │  shipping_     │
              N────┴────N                                         │  company       │
               │         │                                        │  (船公司)      │
          ┌────┴─────┐  └──────┐                                 └────────────────┘
          │shipping  │         │
          │_order    │  shipper_company
          │(订单)    │  (货主公司)
          └────┬─────┘
               │
          ┌────┴──────────┐
          │  order_cargo  │
          │  (订单货物)    │
          └────────────────┘
               │
          ┌────┴────────────────────┐
          │ segment_capacity_usage  │
          │ (航段运力占用)           │
          └─────────────────────────┘
```

### 3.2 各表详解

#### city（城市）

| 字段 | 类型 | 说明 |
|------|------|------|
| city_id | BIGINT PK | 城市编号 |
| city_name | VARCHAR(100) NOT NULL | 城市名称 |
| country | VARCHAR(100) | 国家 |
| country_code | VARCHAR(10) | 国家代码（CN, SG, NL） |
| timezone | VARCHAR(50) | 时区（Asia/Shanghai） |
| latitude | DECIMAL(10,6) | 纬度 |
| longitude | DECIMAL(10,6) | 经度 |
| create_time / update_time / delete_time | DATETIME | 时间戳，软删除 |

#### port（港口）

| 字段 | 类型 | 说明 |
|------|------|------|
| port_id | BIGINT PK | 港口编号 |
| port_name | VARCHAR(200) NOT NULL | 港口名称 |
| port_code | VARCHAR(50) UNIQUE | 联合国口岸代码（CNSHA） |
| city_id | BIGINT FK → city | 所属城市 |
| latitude / longitude | DECIMAL(10,6) | 坐标 |
| port_type | VARCHAR(50) | 港口类型（Sea Port） |
| max_draft_meter | DECIMAL(6,2) | 最大吃水深度 |
| City *City | GORM 关联 | 城市对象 |

#### berth（泊位）

| 字段 | 类型 | 说明 |
|------|------|------|
| berth_id | BIGINT PK | 泊位编号 |
| berth_name | VARCHAR(100) NOT NULL | 泊位名称 |
| port_id | BIGINT FK → port | 所属港口 |
| berth_type | VARCHAR(50) | 泊位类型（Container） |
| draft_meter | DECIMAL(6,2) | 吃水深度 |
| length_meter | DECIMAL(8,2) | 长度 |
| width_meter | DECIMAL(8,2) | 宽度 |
| max_berthing_tonnage | DECIMAL(12,2) | 最大靠泊吨位 |
| functional_zone | VARCHAR(100) | 功能分区 |
| is_available | TINYINT DEFAULT 1 | 0=不可用, 1=可用 |
| Port *Port | GORM 关联 | 港口对象 |

#### shipper_company（货主公司）

| 字段 | 类型 | 说明 |
|------|------|------|
| company_id | BIGINT PK | 公司编号 |
| company_name | VARCHAR(200) NOT NULL | 公司名称 |
| unified_social_credit_code | VARCHAR(50) UK | 统一社会信用代码 |
| legal_representative | VARCHAR(100) | 法定代表人 |
| contact_phone | VARCHAR(50) | 联系电话 |
| address | VARCHAR(500) | 地址 |
| login_username | VARCHAR(100) UNIQUE NOT NULL | 登录用户名 |
| login_password | VARCHAR(255) NOT NULL | bcrypt 哈希 |
| account_status | TINYINT DEFAULT 1 | 0=停用, 1=启用 |

#### shipping_company（船运公司）

| 字段 | 类型 | 说明 |
|------|------|------|
| company_id | BIGINT PK | 公司编号 |
| company_name | VARCHAR(200) NOT NULL | 公司名称 |
| contact_person | VARCHAR(100) | 联系人 |
| contact_phone | VARCHAR(50) | 联系电话 |
| login_username | VARCHAR(100) UNIQUE NOT NULL | 登录用户名 |
| login_password | VARCHAR(255) NOT NULL | bcrypt 哈希 |
| account_status | TINYINT DEFAULT 1 | 0=停用, 1=启用 |

#### admin（管理员）

| 字段 | 类型 | 说明 |
|------|------|------|
| admin_id | BIGINT PK | 管理员编号 |
| username | VARCHAR(100) UNIQUE NOT NULL | 用户名 |
| password | VARCHAR(255) NOT NULL | bcrypt 哈希 |
| real_name | VARCHAR(100) | 姓名 |
| role | TINYINT DEFAULT 2 | 1=超级管理员, 2=普通管理员 |

#### vessel（船舶）

| 字段 | 类型 | 说明 |
|------|------|------|
| vessel_id | BIGINT PK | 船舶编号 |
| vessel_name | VARCHAR(200) NOT NULL | 船名 |
| call_sign | VARCHAR(50) | 船舶呼号 |
| imo_number | VARCHAR(20) UNIQUE NOT NULL | IMO 国际识别号 |
| vessel_type | VARCHAR(50) | 船型（Container Ship） |
| max_deadweight_ton | DECIMAL(12,2) | 最大载重吨位 |
| gross_tonnage | DECIMAL(12,2) | 总吨位 |
| net_tonnage | DECIMAL(12,2) | 净吨位 |
| draft_meter | DECIMAL(6,2) | 吃水深度 |
| speed_knot | DECIMAL(6,2) | 航速 |
| container_teu | INT | 集装箱容量（TEU） |
| is_available | TINYINT DEFAULT 1 | 0=不可用, 1=可用 |
| shipping_company_id | BIGINT FK → shipping_company | 所属船公司 |
| ShippingCompany *ShippingCompany | GORM 关联 | 船公司对象 |

#### shipping_line（航线）

| 字段 | 类型 | 说明 |
|------|------|------|
| line_id | BIGINT PK | 航线编号 |
| line_name | VARCHAR(200) NOT NULL | 航线名称 |
| shipping_company_id | BIGINT FK → shipping_company | 管理船公司 |
| port_sequence | JSON | 途径港口ID数组 [1,2,3] |
| total_distance_nm | DECIMAL(10,2) | 总距离（海里） |
| departure_port_name | VARCHAR(200) | 起运港名称（冗余） |
| destination_port_name | VARCHAR(200) | 目的港名称（冗余） |
| description | TEXT | 描述 |
| ShippingCompany *ShippingCompany | GORM 关联 | 船公司对象 |

#### voyage_berthing（航次靠泊）

| 字段 | 类型 | 说明 |
|------|------|------|
| berthing_id | BIGINT PK | 靠泊编号 |
| line_id | BIGINT FK → shipping_line | 航线 |
| vessel_id | BIGINT FK → vessel | 船舶 |
| voyage_date | DATE NOT NULL | 航次日期 |
| sequence_no | INT NOT NULL | 顺序号（从1递增） |
| port_id | BIGINT FK → port | 停靠港口 |
| berth_id | BIGINT FK → berth | 停靠泊位 |
| planned_arrival_time | DATETIME | 计划到达时间 |
| planned_departure_time | DATETIME | 计划离开时间 |
| actual_arrival_time | DATETIME | 实际到达时间 |
| actual_departure_time | DATETIME | 实际离开时间 |
| draft_at_berthing_meter | DECIMAL(6,2) | 靠泊吃水 |
| is_adjustable | TINYINT DEFAULT 1 | 是否可调整 |
| UK: (line_id, vessel_id, voyage_date, sequence_no) | | 唯一标识一个航次停靠点 |

#### voyage_cargo_note（航次货单）

| 字段 | 类型 | 说明 |
|------|------|------|
| note_id | BIGINT PK | 货单编号 |
| line_id | BIGINT FK → shipping_line | 航线 |
| vessel_id | BIGINT FK → vessel | 船舶 |
| voyage_date | DATE NOT NULL | 航次日期 |
| sequence_no | INT NOT NULL | 顺序号（对应靠泊序号） |
| cargo_name | VARCHAR(200) | 货物名称 |
| cargo_type | VARCHAR(50) | 货物类型 |
| quantity | DECIMAL(18,2) | 数量 |
| weight_ton | DECIMAL(18,3) | 重量（吨） |
| volume_cubic_meter | DECIMAL(18,3) | 体积（立方米） |
| unit_price | DECIMAL(18,2) | 单价 |
| subtotal | DECIMAL(18,2) | 小计 |
| operation_type | VARCHAR(20) | LOAD（装货）/ UNLOAD（卸货） |
| cargo_handled_ton | DECIMAL(18,3) | 装卸货量 |
| cumulative_booked_capacity_ton | DECIMAL(18,3) | 离港累计已订吨位 |

#### shipping_order（订单）— 核心表

| 字段 | 类型 | 说明 |
|------|------|------|
| order_id | BIGINT PK | 订单编号 |
| order_no | VARCHAR(50) UNIQUE NOT NULL | 订单号（ORD{YYYYMMDD}{8hex}） |
| shipper_company_id | BIGINT FK → shipper_company | 货主 |
| city_id | BIGINT FK → city | 关联城市 |
| load_note_id | BIGINT FK → voyage_cargo_note | 装货清单 |
| unload_note_id | BIGINT FK → voyage_cargo_note | 卸货清单 |
| departure_port_id | BIGINT FK → port | 起运港 |
| destination_port_id | BIGINT FK → port | 目的港 |
| expected_departure_date | DATE | 期望起运日 |
| expected_arrival_date | DATE | 期望到达日 |
| total_cost | DECIMAL(18,2) | 总运费（后端计算） |
| shipper_contact | VARCHAR(200) | 发货联系人 |
| consignee_contact | VARCHAR(200) | 收货联系人 |
| payment_status | TINYINT | 0=未支付, 1=已支付, 2=部分支付 |
| order_status | TINYINT | 0=草稿, 1=已确认, 2=运输中, 3=已完成, 4=已取消 |
| total_weight_ton | DECIMAL(18,3) | 货物总重量 |
| total_volume_cubic_meter | DECIMAL(18,3) | 货物总体积 |
| OrderCargos []OrderCargo | 一对多 | 货物明细 |

#### order_cargo（订单货物明细）

| 字段 | 类型 | 说明 |
|------|------|------|
| detail_id | BIGINT PK | 明细编号 |
| order_id | BIGINT FK → shipping_order | 所属订单 |
| cargo_name | VARCHAR(200) | 货物名称 |
| cargo_type | VARCHAR(50) | 货物类型 |
| quantity | DECIMAL(18,2) | 数量 |
| weight_ton | DECIMAL(18,3) | 重量 |
| volume_cubic_meter | DECIMAL(18,3) | 体积 |
| unit_price | DECIMAL(18,2) | 单价 |
| subtotal | DECIMAL(18,2) | 小计 |

#### segment_capacity_usage（航段运力占用）

| 字段 | 类型 | 说明 |
|------|------|------|
| usage_id | BIGINT PK | 占用编号 |
| order_id | BIGINT FK → shipping_order | 订单 |
| line_id | BIGINT FK → shipping_line | 航线 |
| vessel_id | BIGINT FK → vessel | 船舶 |
| voyage_date | DATE NOT NULL | 航次日期 |
| start_port_id | BIGINT FK → port | 起始港口 |
| end_port_id | BIGINT FK → port | 目的港口 |
| occupied_ton | DECIMAL(18,3) NOT NULL | 占用吨位 |
| UK: (order_id, line_id, vessel_id, voyage_date, start_port_id, end_port_id) | | |

### 3.3 软删除策略

| 策略 | 表 |
|------|-----|
| 有 delete_time（软删除） | city, port, berth, shipper_company, shipping_company, admin, vessel, shipping_line, shipping_order, order_cargo |
| 无 delete_time（物理删除/不删除） | voyage_berthing, voyage_cargo_note, segment_capacity_usage |

### 3.4 索引策略

- 所有外键列创建单列索引
- 软删除表创建 delete_time 索引
- 组合索引：`shipping_order(shipper_company_id, order_status)`
- 组合索引：`voyage_cargo_note(line_id, voyage_date)`
- 组合索引：`segment_capacity_usage(line_id, vessel_id, voyage_date, start_port_id, end_port_id)`
- 唯一索引（含 delete_time）：支持软删除的唯一约束，如 `uk_orderno_delete(order_no, delete_time)`

---

## 4. 技术栈

| 类别 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.25.4 |
| HTTP 框架 | Gin | v1.12.0 |
| ORM | GORM + MySQL 驱动 | v1.31.1 / v1.6.0 |
| 数据库 | MySQL | 5.7+ / 8.0+ |
| JWT | golang-jwt | v5.3.1 |
| WebSocket | gorilla/websocket | v1.5.3 |
| Excel | excelize | v2.10.1 |
| 缓存 | go-cache | v2.1.0 |
| 密码 | bcrypt（golang.org/x/crypto） | v0.53.0 |
| 日志 | slog（标准库）+ lumberjack | |
| 验证 | go-playground/validator | v10.30.3 |
| Swagger | swaggo/gin-swagger | v1.6.1 |
| ID 生成 | sony/sonyflake | v1.3.0 |

---

## 5. 项目结构

```
backend/
├── app/
│   └── main.go                        # 程序入口，组件装配
├── cmd/
│   └── seed/main.go                   # 种子数据脚本
├── internal/                          # 私有业务代码
│   ├── model/                         # GORM 实体（13个）
│   │   ├── city.go
│   │   ├── port.go
│   │   ├── berth.go
│   │   ├── shipper_company.go
│   │   ├── shipping_company.go
│   │   ├── admin.go
│   │   ├── vessel.go
│   │   ├── shipping_line.go
│   │   ├── voyage_berthing.go
│   │   ├── voyage_cargo_note.go
│   │   ├── shipping_order.go
│   │   ├── order_cargo.go
│   │   └── segment_capacity_usage.go
│   ├── dao/                           # 数据访问层（接口+实现）
│   │   ├── dao.go                     # NotDeleted scope
│   │   ├── city_dao.go
│   │   ├── port_dao.go
│   │   ├── berth_dao.go
│   │   ├── shipper_company_dao.go
│   │   ├── shipping_company_dao.go
│   │   ├── admin_dao.go
│   │   ├── vessel_dao.go
│   │   ├── shipping_line_dao.go
│   │   ├── voyage_berthing_dao.go
│   │   ├── voyage_cargo_note_dao.go   # FindByPortAndOp, AddCumulativeCapacity
│   │   ├── shipping_order_dao.go
│   │   ├── order_cargo_dao.go
│   │   └── segment_capacity_usage_dao.go
│   ├── biz/                           # 领域逻辑层（无状态）
│   │   ├── container.go               # BizContainer 聚合
│   │   ├── port_sequence_parser.go    # JSON → []int64
│   │   ├── segment_calculator.go      # 计算相邻港口对
│   │   ├── capacity_checker.go        # 运力校验
│   │   ├── cost_calculator.go         # 费用汇总
│   │   ├── order_no_generator.go      # ORD{日期}{8hex}
│   │   ├── order_state_machine.go     # 状态机
│   │   ├── voyage_recommender.go      # 航次推荐算法
│   │   └── errors.go                  # 业务错误定义
│   ├── service/                       # 应用服务层
│   │   ├── common.go                  # Logger, 锁工具
│   │   ├── order_service.go           # 订单核心服务
│   │   ├── voyage_service.go          # 航次推荐服务
│   │   ├── company_service.go         # 货主/船公司 注册登录改密
│   │   ├── admin_service.go           # 管理员 注册登录改密
│   │   ├── port_service.go            # 港口查询
│   │   ├── vessel_service.go          # 船舶查询
│   │   ├── shipping_line_service.go   # 航线查询
│   │   ├── import_export_service.go   # Excel 导入导出
│   │   ├── notification_service.go    # 通知（内存+email/SMS）
│   │   ├── report_service.go          # 报表
│   │   └── websocket_service.go       # WS 推送
│   ├── handler/                       # HTTP 控制器
│   │   ├── handler.go                 # Handlers 聚合
│   │   ├── auth.go                    # 登录/刷新
│   │   ├── company.go                 # 货主/船公司 注册改密
│   │   ├── admin.go                   # 管理员 创建改密
│   │   ├── order.go                   # 订单 CRUD
│   │   ├── voyage.go                  # 航次推荐
│   │   ├── port.go                    # 港口查询
│   │   ├── vessel.go                  # 船舶查询
│   │   ├── shipping_line.go           # 航线查询
│   │   ├── import_export.go           # Excel 导入导出
│   │   ├── notification.go            # 通知管理
│   │   └── report.go                  # 报表
│   └── notify/                        # 邮件/短信发送（新增）
│       ├── email.go                   # SMTP 发送
│       ├── sms.go                     # SMS 接口（console/aliyun/tencent）
│       └── provider.go                # Provider 聚合
├── net/                               # HTTP 基础设施
│   ├── middleware/
│   │   ├── auth.go                    # JWT 认证 + 角色校验
│   │   ├── cors.go                    # CORS
│   │   ├── logger.go                  # 请求日志
│   │   ├── rate_limit.go              # 令牌桶限流
│   │   └── recovery.go                # panic 恢复
│   ├── protect/
│   │   ├── headers.go                 # 安全响应头
│   │   ├── honeypot.go                # 蜜罐陷阱
│   │   ├── ip_blocklist.go            # IP 黑名单
│   │   └── request_guard.go           # 请求守卫
│   ├── router/
│   │   └── router.go                  # 路由注册
│   └── websocket/
│       ├── hub.go                     # 连接管理
│       ├── client.go                  # 读写泵
│       └── handler.go                 # WS 升级 + 推送
├── pkg/                               # 共享基础设施
│   ├── config/config.go               # YAML + 环境变量
│   ├── logger/logger.go               # 日志初始化
│   ├── database/mysql.go              # MySQL 连接池
│   ├── cache/cache.go                 # 内存缓存
│   ├── jwt/jwt.go                     # JWT 令牌
│   ├── crypto/crypto.go               # bcrypt + AES + MD5
│   ├── errors/errors.go               # 错误码
│   ├── response/response.go           # 统一响应
│   ├── validator/validator.go         # 参数校验
│   ├── excel/excel.go                 # Excel 读写
│   ├── timeutil/timeutil.go           # 时间工具
│   ├── idgen/idgen.go                 # ID 生成器
│   └── fileutil/fileutil.go           # 文件操作
├── sql/
│   └── tables_mysql.sql               # DDL 建表脚本
├── docs/                              # Swagger 文档
├── config.yaml                        # 配置文件
├── go.mod / go.sum                    # Go 模块文件
├── project_log/                       # 更新日志
│   ├── 6.25.txt                       # 初始版本说明
│   └── 7.2.txt                        # 本次更新说明
└── README.md                          # 本文档
```

---

## 6. 模块架构

### 6.1 分层架构

```
         ┌───────────────────────────────────────────┐
         │           app/main.go                     │
         │     入口：配置 → 数据库 → JWT → DAO       │
         │     → Biz → Service → Handler → 路由      │
         └───────────────────────────────────────────┘
                            │
   ┌───────────────────────────────────────────────────────────┐
   │                     net/ — 网络层                         │
   │  ┌──────────────┐ ┌──────────────┐ ┌────────┐ ┌──────┐  │
   │  │  middleware   │ │   protect    │ │ router │ │  WS  │  │
   │  │ 认证/日志/限流│ │安全/蜜罐/守卫│ │路由注册│ │ 推送 │  │
   │  └──────────────┘ └──────────────┘ └────────┘ └──────┘  │
   └───────────────────────────────────────────────────────────┘
                            │
   ┌───────────────────────────────────────────────────────────┐
   │               internal/ — 业务层                          │
   │                                                          │
   │   handler  →  service  →  biz  →  dao  →  model          │
   │  (控制器)     (服务)     (逻辑)   (数据)   (实体)        │
   │                                                          │
   │   notify — 邮件/短信发送                                  │
   └───────────────────────────────────────────────────────────┘
                            │
   ┌───────────────────────────────────────────────────────────┐
   │               pkg/ — 基础设施层                           │
   │  config / logger / database / jwt / crypto /              │
   │  errors / response / validator / cache / excel            │
   └───────────────────────────────────────────────────────────┘
```

### 6.2 各模块职责

#### pkg/ — 基础设施层

| 模块 | 文件 | 职责 |
|------|------|------|
| config | config.go | 读取 config.yaml + 环境变量覆盖 |
| logger | logger.go | 初始化 slog，终端+文件双输出 |
| database | mysql.go | GORM 连接 MySQL，连接池配置，健康检查 |
| jwt | jwt.go | HMAC-SHA256 双令牌（15min + 7d） |
| crypto | crypto.go | bcrypt 哈希/校验 + AES-GCM 加解密 + MD5 |
| errors | errors.go | 9 个统一错误码 0~2002 |
| response | response.go | `{code, message, data}` 标准响应 |
| validator | validator.go | struct tag 校验 |
| cache | cache.go | go-cache 封装，支持前缀删除 |
| excel | excel.go | Excelize 读写封装 |
| timeutil | timeutil.go | 日期解析/格式化 |
| idgen | idgen.go | Sonyflake 分布式 ID |

#### net/middleware/ — 中间件链（按注册顺序）

| 顺序 | 中间件 | 作用 |
|------|--------|------|
| 1 | Logger | 记录请求方法/路径/状态码/延迟/IP/UA |
| 2 | Recovery | panic 恢复 + 堆栈日志 |
| 3 | CORS | 允许所有来源 |
| 4 | SecurityHeaders | X-Content-Type-Options / X-Frame-Options / X-XSS-Protection / CSP |
| 5 | Honeypot | 拦截 /wp-admin /phpmyadmin /.env 等扫描路径 |
| 6 | IPBlocklist | 配置中的黑名单 IP |
| 7 | RequestGuard | 限制请求体 4MB / URL 2048 / 方法白名单 |
| 8 | RateLimit | 令牌桶 100tokens/s, burst=20, 429 |
| 9 | RequireAuth | JWT 令牌验证 |
| 10 | RequireRole | 角色校验（admin 组） |

#### net/websocket/ — WebSocket 实时推送

```
Hub (全局单例)
├── clients    map[*Client]bool
├── register   chan *Client
├── unregister chan *Client
├── broadcast  chan []byte
├── stop       chan struct{}
└── Run()      — 无限 select 循环

Client (每个连接)
├── conn *websocket.Conn
├── send chan []byte (buffer=256)
├── userID / role
├── ReadPump()   — 读消息 + pong (60s)
├── WritePump()  — 写消息 + ping (54s)
└── IsActive()

对外接口：
  PushToUser(userID, role, message)  — 定向推送
  ShutdownHub()                      — 优雅关闭
```

#### internal/biz/ — 领域逻辑层（无状态纯函数）

| 组件 | 输入 | 输出 | 算法 |
|------|------|------|------|
| PortSequenceParser | `"[1,2,3]"` | `[]int64{1,2,3}` | json.Unmarshal |
| SegmentCalculator | portIDs, startID, endID | `[][2]int64` | 返回中间所有相邻港口对 |
| CapacityChecker | segments, maxWeight, occupiedGetter, newWeight | `(bool, remaining)` | min(max - used - new) >= 0 |
| CostCalculator | items | `(totalWeight, totalVolume, subtotals)` | 遍历累加 |
| OrderNoGenerator | — | `ORD{YYYYMMDD}{8hex}` | 日期 + 随机 hex |
| OrderStateMachine | oldStatus, newStatus | error | 状态转换规则表 |
| VoyageRecommender | startPort, endPort, requiredTon | `[]Recommendation` | 瓶颈容量算法 + 缓存 1min |

#### internal/dao/ — 数据访问层

每个 DAO 提供接口和实现，关键自定义查询：

```go
// 用于订单创建时定位装卸货点
FindByPortAndOp(lineID, vesselID int64, voyageDate string, portID int64, opType string)
    → JOIN voyage_berthing ON (line_id, vessel_id, voyage_date, sequence_no)
    → WHERE berthing.port_id = ? AND note.operation_type = ?

// 事务内更新累计已订吨位
AddCumulativeCapacity(tx *gorm.DB, noteID int64, deltaTon float64)
    → UPDATE voyage_cargo_note SET cumulative_booked_capacity_ton = cumulative_booked_capacity_ton + delta

// 查询航段已占容量
GetOccupiedTons(lineID, vesselID int64, voyageDate string, startPortID, endPortID int64) (float64, error)
    → SELECT COALESCE(SUM(occupied_ton), 0) FROM segment_capacity_usage WHERE ...

// 全局软删除过滤
NotDeleted(db) *gorm.DB
    → db.Where("delete_time IS NULL")
```

#### internal/service/ — 应用服务层

| 服务 | 核心方法 | 事务/编排 |
|------|---------|-----------|
| **OrderService** | `CreateOrder` | GET_LOCK + 运费计算 + 运力校验 + 多表事务写入 + 缓存清除 + WS 推送 |
| | `CancelOrder` | 锁行 + 释放运力 + 软删除订单/货物 + 物理删除运力占用 |
| | `UpdateOrderStatus` | 状态机校验 + 更新 + WS 推送 |
| | `GetOrderByID` | 预加载货物/清单 |
| | `GetOrderTracking` | 查询装卸时间/靠泊时间/船名/航线名 |
| **VoyageService** | `RecommendVoyages` | 遍历航线 → 解析序列 → 查找航次 → 瓶颈容量 → 排序 → 缓存 |
| **CompanyService** | `Register` | bcrypt 哈希 → 创建 |
| (shipper/shipping) | `Login` | 查用户 → bcrypt 校验 → 检查状态 |
| | `UpdatePassword` | 校验旧密码 → bcrypt 新密码 → 更新 |
| **AdminService** | 同上 | 同上 |
| **PortService** | `List` / `GetByID` | 缓存 10 分钟 |
| **VesselService** | `List` / `GetByID` | 缓存 10 分钟 |
| **ShippingLineService** | `List` / `GetByID` / `GetPortSequence` | — |
| **ImportExportService** | 8 个端点 | Excelize 读写 |
| **NotificationService** | `Send` | 内存存储 + 可选 email/SMS 异步发送 |
| | `GetUserNotifications` | 内存分页查询 |
| | `MarkAsRead` | 标记已读 |
| **ReportService** | `OrderStatistics` | 按日期范围聚合 |
| | `VoyageUtilization` | 已占吨位 / 最大载重 |
| **WebSocketService** | `PushOrderStatusUpdate` | Hub 广播 |

#### internal/notify/ — 邮件/短信发送

| 文件 | 功能 |
|------|------|
| email.go | SMTP 发送（net/smtp），支持纯文本和 HTML |
| sms.go | SMS 接口 + 三种实现：console（仅日志）、aliyun、tencent |
| provider.go | Provider 聚合 |

设计原则：
- 不新增任何数据库表
- 通知内存存储不变（现有机制）
- email/phone 通过 `Notification.Data` 字段传递
- SMTP/SMS 均异步发送，不阻塞主流程
- 配置真实 SMTP 后即可发真实邮件

---

## 7. 核心功能详解

### 7.1 订单管理

#### 创建订单（最复杂流程）

```
1. 校验货物列表非空
2. CostCalculator.Calculate        → 汇总总重量/体积/小计
3. VesselDAO.GetByID               → 获取 max_deadweight_ton
4. ShippingLineDAO.GetByID         → 获取 port_sequence + total_distance_nm
5. 运费计算                          → totalWeight × totalDistance × 0.05 × cargoTypeFactor
6. PortSequenceParser.Parse        → JSON → []int64
7. SegmentCalculator.Calculate     → 起止港之间所有相邻港对
8. VoyageCargoNoteDAO.FindByPortAndOp → 查找 LOAD/UNLOAD 清单
9. 事务：
   a. GET_LOCK("voyage_{line}_{vessel}_{date}", 10)
   b. SELECT FOR UPDATE 锁定航段
   c. CapacityChecker.Check        → 校验 min(max - used - new) >= 0
   d. OrderNoGenerator.Generate    → ORD{YYYYMMDD}{8hex}
   e. INSERT shipping_order        → status=1(已确认), payment=0(未支付)
   f. INSERT order_cargo           → 批量
   g. INSERT segment_capacity_usage → 每个航段一条
   h. 更新 LOAD/UNLOAD 清单的 cumulative_booked_capacity_ton
   i. RELEASE_LOCK
10. cache.DeletePrefix("voyage_rec:")  → 清除推荐缓存
```

#### 取消订单

```
1. SELECT FOR UPDATE 锁行查订单 + 预加载 LOAD/UNLOAD 清单
2. 校验 status != 4（已取消）
3. GET_LOCK("voyage_{line}_{vessel}_{date}")
4. LOAD/UNLOAD 清单的 cumulative_booked_capacity_ton 减去订单重量
5. 软删除 shipping_order（设置 delete_time）
6. 软删除 order_cargo
7. 物理删除 segment_capacity_usage
8. 清除推荐缓存
```

#### 状态更新 + WebSocket 推送

```
1. OrderStateMachine.Transition(oldStatus, newStatus)
   允许转换：
     0→1（草稿→已确认）  0→4（草稿→取消）
     1→2（已确认→运输中） 1→4（已确认→取消）
     2→3（运输中→已完成） 2→4（运输中→取消）
2. 更新数据库
3. wsSvc.PushOrderStatusUpdate → WebSocket 推送
   {"type":"order_status_update","order_id":X,"status":Y,"timestamp":Z}
```

### 7.2 航次推荐

```
GET /api/v1/voyages/recommend?start_port_id=X&end_port_id=Y&required_ton=Z

算法：
1. 遍历所有未删除的航线
2. 解析 port_sequence → []int64
3. 查找所有不同 (vessel_id, voyage_date) 航次
4. 对每个航次：
   a. 计算起止港之间所有航段
   b. 查询每个航段的已占吨位 SUM(occupied_ton)
   c. 瓶颈剩余 = MIN(max_deadweight - 已占吨位)
5. 过滤瓶颈剩余 >= 需求吨位的航次
6. 按瓶颈剩余降序排序
7. 缓存 1 分钟（key: voyage_rec:{start}:{end}:{ton}）
8. 订单创建/取消时清除缓存
```

### 7.3 邮件/短信通知

```
管理员发送通知 POST /api/v1/admin/notifications
  {
    "user_id": 1,
    "role": "shipper",
    "type": "order_created",
    "title": "Order Confirmed",
    "content": "Your order has been created",
    "data": {
      "email": "user@example.com",     // ← 可选，触发真实邮件发送
      "phone": "+8613800000000"         // ← 可选，触发真实短信发送
    }
  }

发送流程：
1. 通知写入内存存储（现有机制）
2. 如果 data.email 存在 → SMTP 异步发邮件
3. 如果 data.phone 存在 → SMS 异步发短信
4. 发送失败仅日志记录，不阻塞主流程

配置 SMTP（config.yaml）：
  notify:
    email:
      smtp_host: "smtp.example.com"
      smtp_port: 587
      username: "noreply@example.com"
      password: "your-password"
      from_addr: "noreply@example.com"
      from_name: "MTS System"
    sms:
      provider: "console"   // console/aliyun/tencent
```

---

## 8. 请求完整链路示例（创建订单）

```
客户端 POST /api/v1/orders
  │
  ├── net/middleware/ (8个全局中间件)
  │   ├── Logger           — 记录请求日志
  │   ├── Recovery         — panic 保护
  │   ├── CORS             — 跨域允许
  │   ├── SecurityHeaders  — 安全响应头
  │   ├── Honeypot         — 扫描器拦截
  │   ├── IPBlocklist      — IP 黑名单
  │   ├── RequestGuard     — 请求体/URL 限制
  │   └── RateLimit        — 令牌桶限流
  │
  ├── net/middleware/auth.go
  │   └── RequireAuth()    — JWT 解析校验
  │                          → c.Set("user_id", ...)
  │                          → c.Set("role", ...)
  │
  ├── internal/handler/order.go
  │   └── CreateOrder()
  │       ├── c.ShouldBindJSON(&req)      — 解析请求体
  │       ├── validator.Validate(req)     — 参数校验
  │       ├── JWT 校验 shipper_company_id  — 越权检查
  │       └── h.svc.CreateOrder(ctx, req) — 调用服务层
  │
  ├── internal/service/order_service.go
  │   ├── CostCalculator.Calculate        → 汇总重量/体积
  │   ├── VesselDAO.GetByID               → 获取船舶载重
  │   ├── ShippingLineDAO.GetByID         → 获取航线距离/港口序列
  │   ├── 运费公式                         → weight × distance × 0.05 × 系数
  │   ├── PortSequenceParser.Parse        → JSON → []int64
  │   ├── SegmentCalculator.Calculate     → 计算航段对
  │   ├── VoyageCargoNoteDAO.FindByPortAndOp → 查 LOAD/UNLOAD 清单
  │   │
  │   └── db.Transaction(func(tx) error {
  │       ├── GET_LOCK("voyage_{line}_{vessel}_{date}", 10)
  │       ├── SELECT FOR UPDATE 锁定航段记录
  │       ├── CapacityChecker.Check       → 校验 min(max - used - new) >= 0
  │       ├── OrderNoGenerator.Generate   → ORD{YYYYMMDD}{8hex}
  │       ├── tx.Create(&order)
  │       ├── tx.Create(&cargos)           — 批量插入
  │       ├── tx.Create(&usages)           — 批量插入
  │       └── AddCumulativeCapacity(tx, ...) — 更新清单累计吨位
  │   })
  │
  └── response.Success(c.Writer, order)   → JSON 响应
```

---

## 9. API 接口总览

| 方法 | 路径 | 角色 | 说明 |
|------|------|------|------|
| GET | `/health` | 公开 | 健康检查 |
| GET | `/ws?token=` | 公开 | WebSocket 连接 |
| POST | `/api/v1/auth/login` | 公开 | 登录 |
| POST | `/api/v1/auth/refresh` | 公开 | 刷新令牌 |
| POST | `/api/v1/shipper/register` | 公开 | 货主注册 |
| POST | `/api/v1/shipping/register` | 公开 | 船公司注册 |
| POST | `/api/v1/shipper/password/:id` | JWT | 货主改密 |
| POST | `/api/v1/shipping/password/:id` | JWT | 船公司改密 |
| POST | `/api/v1/orders` | JWT | 创建订单 |
| GET | `/api/v1/orders/:id` | JWT | 订单详情 |
| POST | `/api/v1/orders/:id/cancel` | JWT | 取消订单 |
| PUT | `/api/v1/orders/:id/status` | JWT | 更新状态 |
| GET | `/api/v1/orders` | JWT | 订单列表 |
| GET | `/api/v1/orders/:id/tracking` | JWT | 货物跟踪 |
| GET | `/api/v1/voyages/recommend` | JWT | 航次推荐 |
| GET | `/api/v1/ports` | JWT | 港口列表 |
| GET | `/api/v1/ports/:id` | JWT | 港口详情 |
| GET | `/api/v1/vessels` | JWT | 船舶列表 |
| GET | `/api/v1/vessels/:id` | JWT | 船舶详情 |
| GET | `/api/v1/shipping-lines` | JWT | 航线列表 |
| GET | `/api/v1/shipping-lines/:id` | JWT | 航线详情 |
| GET | `/api/v1/shipping-lines/:id/port-sequence` | JWT | 港口序列 |
| GET | `/api/v1/export/ports` | JWT | 导出港口 Excel |
| POST | `/api/v1/import/ports` | JWT | 导入港口 Excel |
| GET | `/api/v1/export/vessels` | JWT | 导出船舶 Excel |
| POST | `/api/v1/import/vessels` | JWT | 导入船舶 Excel |
| GET | `/api/v1/export/shipping-lines` | JWT | 导出航线 Excel |
| POST | `/api/v1/import/shipping-lines` | JWT | 导入航线 Excel |
| GET | `/api/v1/export/orders` | JWT | 导出订单 Excel |
| GET | `/api/v1/notifications` | JWT | 通知列表 |
| PUT | `/api/v1/notifications/:id/read` | JWT | 标记已读 |
| GET | `/api/v1/reports/orders` | JWT | 订单统计 |
| GET | `/api/v1/reports/voyage-utilization` | JWT | 航次利用率 |
| POST | `/api/v1/admin/register` | admin | 创建管理员 |
| POST | `/api/v1/admin/password/:id` | admin | 管理员改密 |
| POST | `/api/v1/admin/notifications` | admin | 发送通知 |
| GET | `/swagger/*any` | 公开 | Swagger UI |

---

## 10. 环境要求与启动

### 10.1 环境要求

- Go 1.21+
- MySQL 5.7+ / 8.0+
- （可选）Git

### 10.2 启动步骤

```bash
# 1. 进入项目目录
cd backend

# 2. 安装 Go 依赖
go mod tidy

# 3. 创建数据库并执行建表脚本
mysql -u root -p < sql/tables_mysql.sql

# 4. 修改 config.yaml 中的数据库连接信息
#    vim config.yaml

# 5. 插入种子数据（可选，方便测试）
go run ./cmd/seed

# 6. 启动服务
go run ./app

# 7. 验证启动成功（应看到）：
#    server started port=8080
```

### 10.3 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `DB_DSN` | MySQL 连接串 | (config.yaml) |
| `JWT_SECRET` | JWT 签名密钥 | (config.yaml) |
| `SERVER_PORT` | 监听端口 | 8080 |
| `LOG_LEVEL` | 日志级别 | debug |
| `LOG_OUTPUT_PATH` | 日志输出路径 | logs/app.log |
| `AUTO_MIGRATE` | 启用 GORM 自动迁移 | (不设置) |
| `ENABLE_PPROF` | 启用 pprof 端点 | (不设置) |
| `NOTIFY_EMAIL_*` | 邮件配置 | (config.yaml) |
| `NOTIFY_SMS_PROVIDER` | 短信提供商 | (config.yaml) |

---

## 11. 配置说明

详见 `config.yaml`，支持环境变量覆盖。

```yaml
server:
  port: "8080"
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
  level: debug
  format: text
  output_path: "logs/app.log"
  max_size: 100
  max_backups: 10
  max_age: 30
  compress: true

jwt:
  secret: "your-production-secret-key"     # 必须修改
  access_expire: 15m
  refresh_expire: 168h

freight:
  base_rate_per_ton_nm: 0.05               # 元/吨/海里
  cargo_type_factors:
    bulk: 1.0
    container: 1.2
    liquid: 1.1

notify:
  email:
    smtp_host: "smtp.example.com"
    smtp_port: 587
    username: "noreply@example.com"
    password: "your-email-password"
    from_addr: "noreply@example.com"
    from_name: "MTS System"
  sms:
    provider: "console"                     # console/aliyun/tencent
    access_key_id: ""
    access_key_secret: ""
    sign_name: ""
    template_code: ""
```

---

## 12. 测试示例

以下 PowerShell 命令使用已有的种子数据进行测试（已预置 test001/123456, shipping001/123456, admin/admin123）。

### 12.1 货主登录

```powershell
$login = '{"username":"test001","password":"123456","role":"shipper"}'
$resp = Invoke-RestMethod -Uri http://localhost:8080/api/v1/auth/login -Method POST -Body $login -ContentType "application/json"
$token = $resp.data.access_token
$headers = @{Authorization="Bearer $token"}
$resp | ConvertTo-Json
```

### 12.2 船公司登录

```powershell
Invoke-RestMethod -Uri http://localhost:8080/api/v1/auth/login -Method POST -Body '{"username":"shipping001","password":"123456","role":"shipping"}' -ContentType "application/json" | ConvertTo-Json
```

### 12.3 管理员登录

```powershell
Invoke-RestMethod -Uri http://localhost:8080/api/v1/auth/login -Method POST -Body '{"username":"admin","password":"admin123","role":"admin"}' -ContentType "application/json" | ConvertTo-Json
```

### 12.4 查港口列表

```powershell
Invoke-RestMethod -Uri http://localhost:8080/api/v1/ports -Headers $headers | ConvertTo-Json
```

### 12.5 查船舶列表

```powershell
Invoke-RestMethod -Uri http://localhost:8080/api/v1/vessels -Headers $headers | ConvertTo-Json
```

### 12.6 查航线列表

```powershell
Invoke-RestMethod -Uri http://localhost:8080/api/v1/shipping-lines -Headers $headers | ConvertTo-Json
```

### 12.7 航次推荐（上海→鹿特丹，需求100吨）

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/voyages/recommend?start_port_id=1&end_port_id=3&required_ton=100" -Headers $headers | ConvertTo-Json
```

### 12.8 创建订单

```powershell
$order = @{
    shipper_company_id = 1
    city_id = 1
    line_id = 1
    vessel_id = 1
    voyage_date = "2026-07-15"
    start_port_id = 1
    end_port_id = 3
    cargo_items = @(@{
        cargo_name = "Electronics"
        cargo_type = "container"
        quantity = 100
        weight_ton = 50
        volume_cub_m = 250
        unit_price = 200
        subtotal = 20000
    })
    shipper_contact = "Alice Wang"
    consignee_contact = "Bob Li"
} | ConvertTo-Json -Compress

$orderResp = Invoke-RestMethod -Uri http://localhost:8080/api/v1/orders -Method POST -Body $order -ContentType "application/json" -Headers $headers
$orderResp | ConvertTo-Json
```

### 12.9 货物跟踪

```powershell
Invoke-RestMethod -Uri http://localhost:8080/api/v1/orders/1/tracking -Headers $headers | ConvertTo-Json
```

### 12.10 修改订单状态

```powershell
Invoke-RestMethod -Uri http://localhost:8080/api/v1/orders/1/status -Method PUT -Body '{"status":2}' -ContentType "application/json" -Headers $headers | ConvertTo-Json
```

### 12.11 取消订单

```powershell
Invoke-RestMethod -Uri http://localhost:8080/api/v1/orders/1/cancel -Method POST -Headers $headers | ConvertTo-Json
```

### 12.12 管理员发送通知

```powershell
$adminLogin = Invoke-RestMethod -Uri http://localhost:8080/api/v1/auth/login -Method POST -Body '{"username":"admin","password":"admin123","role":"admin"}' -ContentType "application/json"
$adminToken = $adminLogin.data.access_token

$notifBody = @{
    user_id = 1
    role = "shipper"
    type = "order_created"
    title = "Order Confirmed"
    content = "Your order has been created successfully"
    data = @{
        email = "user@example.com"
        phone = "+8613800000000"
    }
} | ConvertTo-Json -Compress

Invoke-RestMethod -Uri http://localhost:8080/api/v1/admin/notifications -Method POST -Body $notifBody -ContentType "application/json" -Headers @{Authorization="Bearer $adminToken"} | ConvertTo-Json
```

### 12.13 查看通知

```powershell
Invoke-RestMethod -Uri http://localhost:8080/api/v1/notifications -Headers $headers | ConvertTo-Json
```

---

## 13. 常见问题

### Q1: 启动时端口被占用

```
server failed error="listen tcp :8080: bind: Only one usage..."
```

杀掉残留进程：

```powershell
Get-Process -Name "app" -ErrorAction SilentlyContinue | Stop-Process -Force
```

### Q2: 登录返回 "invalid credentials"

用户不存在或密码不是 bcrypt 哈希。先用注册接口创建用户：

```powershell
Invoke-RestMethod -Uri http://localhost:8080/api/v1/shipper/register -Method POST -Body '{"company_name":"Test Co","login_username":"test001","password":"123456"}' -ContentType "application/json"
```

或运行种子脚本：

```bash
go run ./cmd/seed
```

### Q3: 订单创建失败 "no LOAD cargo note for start port"

数据库中缺少对应航线的航次靠泊和装卸货单。运行种子脚本插入测试数据：

```bash
go run ./cmd/seed
```

### Q4: 订单运费为 0

检查 `shipping_line` 表的 `total_distance_nm` 是否为 NULL 或 0。

### Q5: WebSocket 返回 401

确保路径是 `/ws`（不是 `/api/v1/ws`），且 token 未过期。

### Q6: 限流返回 429

默认 100/s，burst=20。可调整 `rate_limit.go` 中的配置。

---

## 14. 性能与安全建议

1. **生产环境关闭 debug 模式**：
   ```bash
   set GIN_MODE=release
   ```

2. **JWT 密钥必须修改**：使用至少 32 字节随机字符串

3. **日志配置**：建议输出到文件并开启轮转（compress: true）

4. **多实例部署**需要改造：
   - 内存缓存（go-cache）→ Redis
   - 通知存储（内存）→ 数据库或 Redis
   - 限流器（内存令牌桶）→ Redis + Lua

5. **定期数据库维护**：清理超过 90 天的软删除记录

6. **启用 pprof**：
   ```bash
   set ENABLE_PPROF=true
   ```
   访问 `/debug/pprof`

7. **Swagger 文档**：
   ```bash
   swag init -g app/main.go -o docs
   ```
   访问 `/swagger/index.html`

---

*文档生成日期：2026-07-02*
