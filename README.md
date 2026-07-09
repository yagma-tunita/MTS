# MTS（Maritime Transport System）航运物流管理系统

版本：1.3
生成日期：2026-07-08

---

## 目录

1. [项目简介](#1-项目简介)
2. [需求分析](#2-需求分析)
3. [数据库设计](#3-数据库设计)
4. [技术栈](#4-技术栈)
5. [项目结构](#5-项目结构)
6. [模块架构](#6-模块架构)
7. [核心功能详解](#7-核心功能详解)
8. [请求完整链路示例（创建订单、状态更新、港口访问）](#8-请求完整链路示例)
9. [API 接口总览](#9-api-接口总览)
10. [环境要求与启动](#10-环境要求与启动)
11. [配置说明](#11-配置说明)
12. [测试账号](#12-测试账号)
13. [常见问题](#13-常见问题)
14. [性能与安全建议](#14-性能与安全建议)

---

## 1. 项目简介

MTS 是一个工业化的航运物流管理平台后端，连接**货主（shipper）**和**船公司（shipping）**，提供货运订单的全生命周期管理。

### 核心业务流程

```
货主注册 → 查询航线/航次 → 选择航次下订单
    → 系统自动计算运费 + 校验船舶剩余运力
    → 订单确认 → 船公司安排运输 → 逐港记录到/离港时间
    → 装卸货操作 → 完成
```

### 三种用户角色

| 角色 | 能力 | 注册方式 |
|------|------|---------|
| **shipper**（货主） | 下订单、查订单、查港口/船舶/航线/航次推荐、追踪物流 | 公开注册 |
| **shipping**（船公司） | 管理订单状态、记录港口操作、管理航线/航次 | 公开注册 |
| **admin**（管理员） | 管理所有数据、航线审核、发通知 | 管理员邀请 |

### 设计原则

- **数据隔离**：shipping 角色只能通过 load_note_id → voyage_cargo_note → shipping_line 链路看到本公司订单
- **实际数据手工录入**：所有实际到/离港时间、装卸货操作均由 shipping 公司通过前端手工录入，系统不自动生成
- **航线生命周期**：待审核(0) → 已启用(1) → 已弃用(2)，支持审核工作流
- **订单状态机**：0(待确认) → 1(已确认) → 2(运输中) → 3(已完成)，任何状态可→ 4(已取消)

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

| 模块 | 功能 | 端点 | 说明 |
|------|------|------|------|
| **订单管理** | 创建订单 | `POST /api/v1/orders` | 5步向导提交，自动计算费用和容量校验 |
| | 订单详情 | `GET /api/v1/orders/:id` | 含 Preload 货主/港口/城市/货物 |
| | 取消订单 | `POST /api/v1/orders/:id/cancel` | 事务内释放容量，软删除 |
| | 更新状态 | `PUT /api/v1/orders/:id/status` | 支持选港+实际时间+货物操作 |
| | 订单列表 | `GET /api/v1/orders` | 角色自动过滤，支持 keyword+status+sort |
| | 货物跟踪 | `GET /api/v1/orders/:id/tracking` | 8步数据装配，含船舶/货物/时间线 |
| | 支付 | `POST /api/v1/orders/:id/pay` | 虚拟支付，仅更新支付状态 |
| | 港口访问记录 | `POST /api/v1/orders/:id/port-visit` | 运输中状态记录到/离港+装卸货 |
| **密码修改** | 货主 | `POST /api/v1/shipper/password/:id` | 需验证旧密码 |
| | 船公司 | `POST /api/v1/shipping/password/:id` | 需验证旧密码 |
| **数据查询** | 港口列表/详情 | `GET /api/v1/ports` / `GET /api/v1/ports/:id` | 缓存10分钟 |
| | 船舶列表/详情 | `GET /api/v1/vessels` / `GET /api/v1/vessels/:id` | shipping 角色自动过滤 |
| | 航线列表/详情 | `GET /api/v1/shipping-lines` / `GET /api/v1/shipping-lines/:id` | shipping 角色自动过滤 |
| | 航线港口序列 | `GET /api/v1/shipping-lines/:id/port-sequence` | 解析 JSON 数组 |
| **航次推荐** | 推荐航次 | `GET /api/v1/voyages/recommend` | 4阶段算法+缓存 |
| **航次管理** | 创建航次 | `POST /api/v1/voyages/berthing` | 批量创建靠泊记录 |
| | 我的航次 | `GET /api/v1/voyages/my` | 船公司查看自己的靠泊记录 |
| **靠泊管理** | 更新时间 | `PUT /api/v1/berthings/:id/actual-times` | 更新实际到/离港时间 |
| **导入导出** | 导出港口/船舶/航线/订单 | `GET /api/v1/export/*` | Excel xlsx |
| | 导入港口/船舶/航线 | `POST /api/v1/import/*` | 批量导入 |
| **通知** | 通知列表 | `GET /api/v1/notifications` | 分页查询 |
| | 标记已读 | `PUT /api/v1/notifications/:id/read` | |
| **报表** | 订单统计 | `GET /api/v1/reports/orders` | 日期范围聚合 |
| | 航次利用率 | `GET /api/v1/reports/voyage-utilization` | 载重/已用/利用率 |
| **航线管理** | 删除航线 | `DELETE /api/v1/shipping-lines/:id` | shipping 角色仅置为 deprecated |
| | 重新申请 | `POST /api/v1/shipping-lines/:id/reactivate` | 设置 line_status=0 |

#### 管理员功能（需 role=admin）

| 功能 | 端点 | 说明 |
|------|------|------|
| 管理员列表 | `GET /api/v1/admin/list` | 分页+关键词搜索 |
| 创建管理员 | `POST /api/v1/admin/register` | |
| 修改密码 | `POST /api/v1/admin/password/:id` | |
| 货主管理 | `GET /api/v1/admin/shipper/list` | 列表/更新/删除 |
| 船公司管理 | `GET /api/v1/admin/shipping/list` | 列表/更新/删除 |
| 货物管理 | `GET /api/v1/admin/cargo/list` | 列表+搜索+新建+删除 |
| 航线审核 | `GET /api/v1/admin/shipping-lines/pending` | 待审核列表 |
| | `POST /api/v1/admin/shipping-lines/:id/approve` | 通过→line_status=1 |
| | `POST /api/v1/admin/shipping-lines/:id/deprecate` | 弃用→line_status=2 |
| 发送通知 | `POST /api/v1/admin/notifications` | 给指定用户发通知 |
| CRUD | 城市/港口/船舶/航线 | 完整的增删改查 |

#### WebSocket 实时推送

| 功能 | 路径 | 说明 |
|------|------|------|
| 订单状态推送 | `ws://host/ws?token=<access_token>` | 状态变更时自动推送 |

### 2.2 业务规则

```
1. 订单状态机
   0(待确认) → 1(已确认) → 2(运输中) → 3(已完成)
         ↘ 4(已取消) ←───────────────↙
   每个转换可附带：port_id(指定港口)、actual_time(实际时间)、cargo_operations(货物操作)

2. 航线状态机
   海运公司创建 → 0(待审核) → 管理员通过 → 1(已启用)
                          → 管理员拒绝 → 2(已弃用)
   1(已启用) → 海运公司停用或管理员弃用 → 2(已弃用)
   2(已弃用) → 海运公司重新申请 → 0(待审核) → 管理员通过 → 1(已启用)
             → 管理员恢复 → 1(已启用)
             → 管理员永久删除 → 物理删除

3. 运力校验
   订单占用航段 = 起运港到目的港之间的所有相邻港口对
   每个航段：max_deadweight - 已占吨位 - 新订单吨位 ≥ 0
   任一航段不足则拒绝订单

4. 并发控制
   使用 MySQL GET_LOCK("order_create", 10) + SELECT FOR UPDATE

5. 软删除
   所有业务表通过 delete_time 标记删除，查询过滤 WHERE delete_time IS NULL

6. 货物装载状态计算
   已装船 = Σ(LOAD 操作的 weight_ton)
   已卸货 = Σ(UNLOAD 操作的 weight_ton)
   状态：plan == 0 ? 待装船 : loaded >= plan ? 已装船 : loaded > 0 ? 部分装船 : discharged >= plan ? 已卸货

7. 船舶当前载货量
   current_load = Σ(LOAD weight_ton) - Σ(UNLOAD weight_ton)
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
              ┌────┴──────────────────────────┐
              │    voyage_berthing             │ ←── shipping_line ──┐
              │    (航次靠泊: line_id,vessel_id,│     (航线: shipping  │
              │     voyage_date,sequence_no)   │      _company_id FK) │
              └────────────────────────────────┘                    │
              ┌────────────────────────────────┐                    │
              │   voyage_cargo_note             │ ←──────────────────┘
              │   (航次货物通知单: note_id,      │              ┌─────┴──────────┐
              │    operation_type LOAD/UNLOAD) │              │  shipping_     │
              └────────────────────────────────┘              │  company       │
                                                              │  (船公司)      │
                  N────┴────N                                 └────────────────┘
                   │         │
              ┌────┴─────┐  └──────┐
              │shipping  │         │
              │_order    │  shipper_company
              │(订单:    │  (货主公司)
              │ load_note│
              │ _id FK)  │
              └────┬─────┘
                   │
              ┌────┴──────────┐
              │  order_cargo  │
              │  (订单货物)    │
              └────┬──────────┘
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
| city_name | VARCHAR(100) NOT NULL | 城市名称，ORDER BY |
| country | VARCHAR(100) | 国家 |
| country_code | VARCHAR(10) | 国家代码（CN, SG, NL） |
| timezone | VARCHAR(50) | 时区（Asia/Shanghai） |

#### port（港口）

| 字段 | 类型 | 说明 |
|------|------|------|
| port_id | BIGINT PK | 港口编号 |
| port_name | VARCHAR(200) NOT NULL | 港口名称，ORDER BY |
| port_code | VARCHAR(20) | 联合国口岸代码（CNNSH） |
| city_id | BIGINT FK → city | 所属城市 |
| port_type | VARCHAR(50) | Sea Port / River Port / Inland Port |
| latitude | DECIMAL(10,6) | 纬度 |
| longitude | DECIMAL(10,6) | 经度 |
| max_draft_meter | DECIMAL(6,2) | 最大吃水深度(米) |

#### berth（泊位）

| 字段 | 类型 | 说明 |
|------|------|------|
| berth_id | BIGINT PK | 泊位编号 |
| berth_name | VARCHAR(200) NOT NULL | 泊位名称 |
| port_id | BIGINT FK → port | 所属港口 |
| berth_type | VARCHAR(50) | Container / Bulk |
| draft_meter | DECIMAL(6,2) | 水深(米) |
| length_meter | DECIMAL(10,2) | 长度(米) |
| width_meter | DECIMAL(10,2) | 宽度(米) |
| max_berthing_tonnage | DECIMAL(12,2) | 最大靠泊吨位 |
| functional_zone | VARCHAR(100) | 功能分区 |
| is_available | TINYINT DEFAULT 1 | 0=不可用, 1=可用 |

#### shipper_company（货主公司）

| 字段 | 类型 | 说明 |
|------|------|------|
| company_id | BIGINT PK | 公司编号 |
| company_name | VARCHAR(200) NOT NULL | 公司名称 |
| login_username | VARCHAR(50) UNIQUE NOT NULL | 登录用户名 |
| login_password | VARCHAR(255) NOT NULL | bcrypt 哈希 |
| unified_social_credit_code | VARCHAR(50) | 统一社会信用代码 |
| legal_representative | VARCHAR(50) | 法定代表人 |
| contact_phone | VARCHAR(20) | 联系电话 |
| address | VARCHAR(500) | 地址 |
| account_status | TINYINT DEFAULT 1 | 0=停用, 1=启用 |

#### shipping_company（船运公司）

| 字段 | 类型 | 说明 |
|------|------|------|
| company_id | BIGINT PK | 公司编号 |
| company_name | VARCHAR(200) NOT NULL | 公司名称 |
| login_username | VARCHAR(50) UNIQUE NOT NULL | 登录用户名 |
| login_password | VARCHAR(255) NOT NULL | bcrypt 哈希 |
| contact_person | VARCHAR(50) | 联系人（区别于货主的 legal_representative） |
| contact_phone | VARCHAR(20) | 联系电话 |
| address | VARCHAR(500) | 地址 |
| account_status | TINYINT DEFAULT 1 | 0=停用, 1=启用 |

#### admin（管理员）

| 字段 | 类型 | 说明 |
|------|------|------|
| admin_id | BIGINT PK | 管理员编号 |
| username | VARCHAR(50) UNIQUE NOT NULL | 用户名 |
| password | VARCHAR(255) NOT NULL | bcrypt 哈希 |
| real_name | VARCHAR(50) | 姓名 |
| role | TINYINT DEFAULT 1 | 1=超级管理员, 2=普通管理员 |

#### vessel（船舶）

| 字段 | 类型 | 说明 |
|------|------|------|
| vessel_id | BIGINT PK | 船舶编号 |
| vessel_name | VARCHAR(200) NOT NULL | 船名 |
| call_sign | VARCHAR(20) | 船舶呼号 |
| imo_number | VARCHAR(20) NOT NULL UNIQUE | IMO 国际识别号 |
| vessel_type | VARCHAR(50) | Container Ship / Bulk Carrier / Oil Tanker / General Cargo |
| max_deadweight_ton | DECIMAL(12,3) | 最大载重吨位（容量计算核心字段） |
| gross_tonnage | DECIMAL(12,3) | 总吨位 |
| speed_knot | DECIMAL(5,2) | 航速(节) |
| container_teu | INT | 集装箱容量（TEU） |
| shipping_company_id | BIGINT FK → shipping_company | 所属船公司 |
| is_available | TINYINT DEFAULT 1 | 0=不可用, 1=可用 |

#### shipping_line（航线）

| 字段 | 类型 | 说明 |
|------|------|------|
| line_id | BIGINT PK | 航线编号 |
| line_name | VARCHAR(200) NOT NULL | 航线名称 |
| shipping_company_id | BIGINT FK → shipping_company | 管理船公司 |
| port_sequence | JSON NOT NULL | 途径港口ID数组 [1,6,7,9,13] |
| total_distance_nm | DECIMAL(10,2) | 总距离（海里） |
| departure_port_name | VARCHAR(200) | 起运港名称（冗余） |
| destination_port_name | VARCHAR(200) | 目的港名称（冗余） |
| description | TEXT | 描述 |
| line_status | TINYINT DEFAULT 1 | 0=待审核, 1=已启用, 2=已弃用 |

#### shipping_order（订单）— 核心表

| 字段 | 类型 | 说明 |
|------|------|------|
| order_id | BIGINT PK | 订单编号 |
| order_no | VARCHAR(50) UNIQUE NOT NULL | 订单号（ORD+yyyMMdd+8hex） |
| shipper_company_id | BIGINT FK → shipper_company | 货主 |
| load_note_id | BIGINT FK → voyage_cargo_note | 装货通知单（数据隔离关键链路） |
| unload_note_id | BIGINT FK → voyage_cargo_note | 卸货通知单 |
| departure_port_id | BIGINT FK → port | 起运港 |
| destination_port_id | BIGINT FK → port | 目的港 |
| expected_departure_date | DATE | 期望起运日 |
| expected_arrival_date | DATE | 期望到达日 |
| total_cost | DECIMAL(18,2) | 总运费（后端计算） |
| shipper_contact | VARCHAR(100) | 发货联系人 |
| consignee_contact | VARCHAR(100) | 收货联系人 |
| payment_status | TINYINT | 0=未支付, 1=已支付 |
| order_status | TINYINT | 0=待确认, 1=已确认, 2=运输中, 3=已完成, 4=已取消 |
| total_weight_ton | DECIMAL(18,3) | 货物总重量 |
| total_volume_cubic_meter | DECIMAL(18,3) | 货物总体积 |

#### order_cargo（订单货物明细）

| 字段 | 类型 | 说明 |
|------|------|------|
| detail_id | BIGINT PK | 明细编号 |
| order_id | BIGINT FK → shipping_order | 所属订单 |
| cargo_name | VARCHAR(200) | 货物名称 |
| cargo_type | VARCHAR(50) | bulk / container / liquid |
| quantity | DECIMAL(18,2) | 数量 |
| weight_ton | DECIMAL(18,3) | 重量(吨) |
| volume_cubic_meter | DECIMAL(18,3) | 体积(m³) |
| unit_price | DECIMAL(18,2) | 单价 |
| subtotal | DECIMAL(18,2) | 小计 = weight_ton × unit_price |

#### voyage_berthing（航次靠泊）

| 字段 | 类型 | 说明 |
|------|------|------|
| berthing_id | BIGINT PK | 靠泊编号 |
| line_id | BIGINT FK → shipping_line | 航线 |
| vessel_id | BIGINT FK → vessel | 船舶 |
| voyage_date | DATE NOT NULL | 航次日期 |
| sequence_no | INT NOT NULL | 顺序号（从1递增） |
| port_id | BIGINT FK → port | 停靠港口 |
| berth_id | BIGINT FK → berth NULL | 停靠泊位 |
| planned_arrival_time | DATETIME | 计划到达时间 |
| planned_departure_time | DATETIME | 计划离开时间 |
| actual_arrival_time | DATETIME NULL | 实际到达时间（手工录入） |
| actual_departure_time | DATETIME NULL | 实际离开时间（手工录入） |
| draft_at_berthing_meter | DECIMAL(6,2) | 靠泊吃水 |
| is_adjustable | TINYINT DEFAULT 1 | 是否可调整 |
| UK: (line_id,vessel_id,voyage_date,sequence_no) | | 唯一标识一个停靠点 |

#### voyage_cargo_note（航次货物通知单）

| 字段 | 类型 | 说明 |
|------|------|------|
| note_id | BIGINT PK | 货单编号（被 order.load_note_id 引用） |
| line_id | BIGINT FK → shipping_line | 航线 |
| vessel_id | BIGINT FK → vessel | 船舶 |
| voyage_date | DATE NOT NULL | 航次日期 |
| sequence_no | INT NOT NULL | 顺序号（对应靠泊序号） |
| cargo_name | VARCHAR(200) | 货物名称 |
| cargo_type | VARCHAR(50) | 货物类型 |
| weight_ton | DECIMAL(18,3) | 重量（吨） |
| volume_cubic_meter | DECIMAL(18,3) | 体积（立方米） |
| operation_type | VARCHAR(20) | LOAD（装货）/ UNLOAD（卸货） |
| cargo_handled_ton | DECIMAL(18,3) | 已处理吨数 |
| cumulative_booked_capacity_ton | DECIMAL(18,3) | 离港累计已订吨位 |

#### segment_capacity_usage（航段运力占用）

| 字段 | 类型 | 说明 |
|------|------|------|
| usage_id | BIGINT PK | 占用编号 |
| line_id | BIGINT | 航线 |
| vessel_id | BIGINT | 船舶 |
| voyage_date | DATE NOT NULL | 航次日期 |
| segment_index | INT NOT NULL | 航段索引（从0开始） |
| order_id | BIGINT FK → shipping_order | 订单 |
| weight_ton | DECIMAL(18,3) | 占用重量(吨) |

### 3.3 软删除策略

| 策略 | 表 |
|------|-----|
| 有 delete_time（软删除） | city, port, berth, shipper_company, shipping_company, admin, vessel, shipping_line, shipping_order, order_cargo |
| 无 delete_time（物理/不删） | voyage_berthing, voyage_cargo_note, segment_capacity_usage |

### 3.4 索引策略

| 表 | 索引字段 | 类型 | 目的 |
|------|------|------|------|
| shipping_order | order_no | UNIQUE | 订单号唯一性 |
| shipping_order | shipper_company_id | INDEX | 货主导航查询 |
| shipping_order | load_note_id | INDEX | 海运公司数据隔离 JOIN |
| vessel | imo_number | UNIQUE | IMO 编号唯一性 |
| vessel | shipping_company_id | INDEX | 公司船舶列表 |
| shipping_line | shipping_company_id | INDEX | 公司航线列表 |
| port | city_id | INDEX | 按城市筛选港口 |
| voyate_berthing | (line_id,vessel_id,voyage_date,sequence_no) | UNIQUE COMPOSITE | 靠泊记录唯一性 |
| segment_capacity_usage | (line_id,vessel_id,voyage_date,segment_index) | COMPOSITE | 容量聚合查询 |
| 所有表 | delete_time | INDEX | 逻辑删除过滤 |

### 3.5 范式分析

**1NF**：所有字段为原子值。port_sequence 以 MySQL JSON 类型存储，应用层解析为独立 int64，满足 1NF。

**2NF**：所有非主键字段完全依赖主键。voyage_berthing 四元组业务键保证完全函数依赖。

**3NF**：通过外键替代冗余存储（port.city_id → city.city_name，消除传递依赖）。

---

## 4. 技术栈

| 类别 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.25.4 |
| HTTP 框架 | Gin | v1.12.0 |
| ORM | GORM | v2 |
| 数据库 | MySQL | 8.4+ |
| JWT | golang-jwt | v5 |
| WebSocket | gorilla/websocket | v1.5 |
| Excel | excelize | v2.10 |
| 缓存 | go-cache | v2 |
| 密码 | bcrypt | golang.org/x/crypto |
| 日志 | logrus | |
| 验证 | go-playground/validator | v10 |
| ID 生成 | crypto/rand | 标准库 |
| 配置管理 | Viper | |

---

## 5. 项目结构

```
backend/
├── app/
│   └── main.go                        # 程序入口，组件装配
├── cmd/
│   └── seed/main.go                   # 种子数据脚本（20城市/20港口/40泊位/16船/15航线/10航次/10订单）
├── internal/
│   ├── model/                         # GORM 实体（13个）
│   │   ├── city.go, port.go, berth.go
│   │   ├── shipper_company.go, shipping_company.go, admin.go
│   │   ├── vessel.go, shipping_line.go
│   │   ├── voyage_berthing.go, voyage_cargo_note.go
│   │   ├── shipping_order.go, order_cargo.go
│   │   └── segment_capacity_usage.go
│   ├── dao/                           # 数据访问层
│   │   ├── dao.go                     # NotDeleted scope
│   │   ├── city_dao.go, port_dao.go, berth_dao.go
│   │   ├── shipper_company_dao.go, shipping_company_dao.go, admin_dao.go
│   │   ├── vessel_dao.go, shipping_line_dao.go
│   │   ├── shipping_order_dao.go, order_cargo_dao.go
│   │   └── segment_capacity_usage_dao.go
│   ├── biz/                           # 领域逻辑层（无状态纯函数）
│   │   ├── port_sequence_parser.go    # JSON → []int64
│   │   ├── segment_calculator.go      # 计算相邻港口对
│   │   ├── capacity_checker.go        # 运力校验 MIN(max-used-new) >= 0
│   │   ├── cost_calculator.go         # 费用汇总
│   │   ├── order_no_generator.go      # ORD{日期}{8hex}
│   │   ├── order_state_machine.go     # 状态机 map 转换表
│   │   └── errors.go                  # 业务错误
│   ├── service/                       # 应用服务层
│   │   ├── common.go                  # Logger, 分页工具, PtrInt8
│   │   ├── order_service.go           # 订单核心服务（创建/取消/状态更新/追踪/港口访问）
│   │   ├── voyage_service.go          # 航次推荐服务
│   │   ├── port_service.go            # 港口查询+缓存
│   │   ├── vessel_service.go          # 船舶查询
│   │   ├── shipping_line_service.go   # 航线查询
│   │   ├── cargo_service.go           # 货物管理
│   │   ├── company_service.go         # 货主/船公司注册改密
│   │   ├── admin_service.go           # 管理员管理
│   │   ├── report_service.go          # 报表
│   │   ├── notification_service.go   # 通知
│   │   └── websocket_service.go       # WS 推送
│   └── handler/                       # HTTP 处理器层
│       ├── handler.go                 # Handlers 聚合
│       ├── auth.go                    # 登录/刷新
│       ├── company.go                 # 货主/船公司
│       ├── admin.go                   # 管理员
│       ├── order.go                   # 订单 CRUD + 状态更新 + 追踪 + 港口访问 + 支付
│       ├── voyage.go                  # 航次推荐/创建
│       ├── berthing.go                # 靠泊管理
│       ├── port.go, vessel.go, shipping_line.go
│       ├── cargo.go                   # 货物管理
│       ├── city.go                    # 城市管理
│       ├── import_export.go           # Excel 导入导出
│       ├── notification.go            # 通知
│       └── report.go                  # 报表
├── net/
│   ├── middleware/
│   │   ├── auth.go                    # JWT 认证 + 角色校验
│   │   ├── cors.go                    # CORS 跨域
│   │   ├── logger.go                  # 请求日志
│   │   ├── rate_limit.go              # 令牌桶限流
│   │   └── recovery.go                # panic 恢复
│   ├── protect/
│   │   ├── headers.go                 # 安全响应头
│   │   ├── honeypot.go                # 蜜罐陷阱
│   │   ├── ip_blocklist.go            # IP 黑名单
│   │   └── request_guard.go           # 请求守卫
│   ├── router/
│   │   └── router.go                  # 路由注册（约60条）
│   └── websocket/
│       ├── hub.go                     # 连接管理
│       ├── client.go                  # 读写泵
│       └── handler.go                 # WS 升级 + 推送
├── pkg/
│   ├── config/config.go               # YAML + 环境变量
│   ├── database/mysql.go              # MySQL 连接池
│   ├── cache/cache.go                 # go-cache 缓存
│   ├── jwt/jwt.go                     # JWT 令牌
│   ├── crypto/crypto.go               # bcrypt
│   ├── errors/errors.go               # 错误码
│   ├── response/response.go           # 统一响应
│   ├── validator/validator.go         # 参数校验
│   ├── excel/excel.go                 # Excel 读写
│   └── idgen/idgen.go                 # ID 生成
├── sql/
│   └── tables_mysql.sql               # DDL 建表脚本
├── config.yaml                        # 配置文件
├── go.mod / go.sum
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
   │   handler  →  service  →  biz  →  dao  →  model          │
   │  (控制器)     (服务)     (逻辑)   (数据)   (实体)        │
   └───────────────────────────────────────────────────────────┘
                            │
   ┌───────────────────────────────────────────────────────────┐
   │               pkg/ — 基础设施层                           │
   │  config / database / jwt / crypto / cache / errors       │
   │  response / validator / excel / idgen                    │
   └───────────────────────────────────────────────────────────┘
```

### 6.2 各模块职责

#### pkg/ — 基础设施层

| 模块 | 职责 |
|------|------|
| config | 读取 config.yaml + 环境变量覆盖 |
| database | GORM MySQL 连接池，健康检查 |
| jwt | HMAC-SHA256 双令牌（24h access + 7d refresh） |
| crypto | bcrypt 哈希/校验 |
| errors | 9 个错误码（1000~5000） |
| response | {code, message, data, meta} 标准响应 |
| validator | struct tag 校验 |
| cache | go-cache 封装，支持泛型 Get[T]，前缀删除 ClearPrefix |
| excel | Excelize 读写封装 |

#### net/middleware/ — 中间件链（按注册顺序）

| 顺序 | 中间件 | 作用 |
|------|--------|------|
| 1 | Logger | 记录方法/路径/状态码/延迟/IP |
| 2 | Recovery | panic 恢复 + 堆栈日志 |
| 3 | CORS | 允许跨域 |
| 4 | SecurityHeaders | X-Content-Type-Options / X-Frame-Options / CSP |
| 5 | Honeypot | 拦截爬虫路径 |
| 6 | IPBlocklist | IP 黑名单 |
| 7 | RequestGuard | 限制请求体/URL/方法 |
| 8 | RateLimit | 令牌桶 100tokens/s, 429 |
| 9 | RequireAuth | JWT 验证 → 注入 user_id/role |
| 10 | RequireRole | 角色校验（admin 组） |

#### net/websocket/ — WebSocket 实时推送

```
Hub (全局单例)
├── clients    map[*Client]bool
├── register   chan *Client
├── unregister chan *Client
├── broadcast  chan []byte
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
  PushOrderStatusUpdate(userID, role, orderID, newStatus) — 订单状态变更推送
```

#### internal/biz/ — 领域逻辑层

| 组件 | 输入 | 输出 | 算法 |
|------|------|------|------|
| PortSequenceParser | "[1,2,3]" | []int64{1,2,3} | json.Unmarshal |
| SegmentCalculator | portIDs, startID, endID | [][2]int64 | 返回中间所有相邻港口对 |
| CapacityChecker | segments, maxWeight, newWeight | (bool, remaining) | MIN(max - used - new) >= 0 |
| CostCalculator | items | (total, subtotals) | 遍历累加 |
| OrderNoGenerator | — | ORD{YYYYMMDD}{8hex} | 日期 + crypto/rand |
| OrderStateMachine | oldStatus, newStatus | error | map 转换表 O(1) |

#### internal/dao/ — 数据访问层关键方法

```go
// 全局软删除过滤
NotDeleted(db) *gorm.DB → db.Where("delete_time IS NULL")

// 分页通用封装
Paginate(query, req, model) → (paginatedQuery, total, error)
  // 支持 sort_by / sort_order 排序，keyword LIKE 搜索，status 筛选

// Map-based Updates（避免 NULL 覆盖）
updates := map[string]interface{}{}
if ptr != nil { updates["field"] = *ptr }
db.Model(&T{}).Updates(updates)
```

#### internal/service/ — 应用服务层

| 服务 | 核心方法 | 事务/编排 |
|------|---------|-----------|
| **OrderService** | `CreateOrder` | 6步事务：费用计算→船舶校验→航线解析→航段计算→cargo_note→GET_LOCK+FOR UPDATE |
| | `UpdateOrderStatus` | 状态机+DAO+靠泊时间+货物操作+WS推送 |
| | `RecordPortVisit` | 运输中状态：更新指定港口的到/离港时间+装卸货操作 |
| | `CancelOrder` | 事务：锁行→释放容量→软删除 |
| | `GetOrderTracking` | 8步数据装配：订单→LoadNote→vessel/line→VoyageBerthing→VoyageCargoNote→货物状态→船位→响应 |
| **VoyageService** | `Recommend` | 4阶段：航线过滤→航段计算→瓶颈容量→排序缓存 |
| **PortService** | `List` / `GetByID` | 缓存 10 分钟，更新时清除缓存 |
| **ShippingLineService** | `ListLines` / `ListLinesByCompany` | 支持 line_status 状态过滤 |
| **CargoService** | `ListAllCargos` | 关键词搜索+分页 |
| **ReportService** | `OrderStatistics` | 日期范围聚合 |
| | `VoyageUtilization` | 已占吨位/最大载重 |

---

## 7. 核心功能详解

### 7.1 订单管理

#### 创建订单（最复杂流程）

```
1. 校验货物列表非空
2. CostCalculator.Calculate        → 汇总总重量/体积
3. VesselDAO.GetByID               → 获取 max_deadweight_ton
4. ShippingLineDAO.GetByID         → 获取 port_sequence
5. 运费计算                        → total = Σ(weight × rate_by_type)
6. PortSequenceParser.Parse        → JSON → []int64
7. SegmentCalculator.Calculate     → 起止港之间所有相邻港对
8. voyage_cargo_note 加载          → 查找 LOAD/UNLOAD 清单
9. 事务：
   a. GET_LOCK("order_create", 10)
   b. SELECT FOR UPDATE 锁定航段
   c. CapacityChecker.Check        → 校验 min(max - used - new) >= 0
   d. OrderNoGenerator.Generate    → ORD{YYYYMMDD}{8hex}
   e. INSERT shipping_order
   f. INSERT order_cargo           → 批量
   g. INSERT segment_capacity_usage → 每航段一条
   h. 更新 LOAD/UNLOAD 清单的 cumulative_booked_capacity_ton
   i. RELEASE_LOCK
10. cache.ClearPrefix("voyage:recommend:")
```

#### 更新订单状态（ship→transit→complete，带港口和货物操作）

```
PUT /api/v1/orders/:id/status
{
  "status": 2,                          // 目标状态
  "port_id": 1,                         // 操作的港口ID
  "actual_time": "2026-05-01 18:00:00", // 实际时间
  "cargo_operations": [                 // 货物操作
    {"cargo_name":"铁矿石","cargo_type":"bulk","weight_ton":50,"operation":"LOAD"}
  ],
  "notes": "装货完成"                     // 备注
}

处理流程：
1. 加载订单 + LoadNote
2. 状态机校验合法性
3. DAO 更新 order_status
4. 根据 port_id 更新对应 voyage_berthing 的 actual_departure_time 或 actual_arrival_time
5. 处理 cargo_operations：查找或创建 voyage_cargo_note（按 cargo_name+operation_type 去重）
6. WebSocket 推送状态变更
```

#### 港口访问记录（运输中状态下的多次更新）

```
POST /api/v1/orders/:id/port-visit
{
  "port_id": 6,                         // 新加坡港
  "actual_arrival": "2026-05-05 08:00:00",
  "actual_departure": "2026-05-06 18:00:00",
  "cargo_operations": [
    {"cargo_name":"橡胶","cargo_type":"bulk","weight_ton":30,"operation":"LOAD"}
  ]
}

处理流程：
1. 校验订单状态为 2（运输中）
2. 查找该港口的 voyage_berthing 获取 sequence_no
3. 更新 actual_arrival_time / actual_departure_time
4. 处理 cargo_operations：去重创建/更新 voyage_cargo_note
```

#### 物流追踪（GetOrderTracking）

```
GET /api/v1/orders/:id/tracking

返回结构：
{
  "order_id": 1,
  "order_no": "ORD20260501a1b2c3d4",
  "order_status": 2,           // 当前状态
  "vessel_name": "远洋号",     // 船舶信息
  "vessel_type": "Bulk Carrier",
  "vessel_capacity": 300000,   // max_deadweight_ton
  "vessel_teu": 3000,
  "vessel_speed": 14.5,
  "vessel_current_load": 150,  // 当前载货量(总LOAD-总UNLOAD)
  "line_name": "中国-南美航线",
  "cargo_summary": [           // 货物装载状态
    {"cargo_name":"铁矿石","weight_ton":150,"loaded_ton":150,"discharged":50,
     "status":"partial"}
  ],
  "stops": [                   // 航次时间线
    {"port_id":1,"port_name":"广州","sequence_no":1,
     "planned_arrival":"...","planned_departure":"...",
     "actual_arrival":null,"actual_departure":"...",
     "status":"completed",
     "cargo_operations":[{"cargo_name":"铁矿石","weight_ton":150,"operation":"LOAD"}]}
  ],
  "current_stop_index": 2      // 当前船位索引(第一个未到港的站位)
}
```

### 7.2 航线推荐算法

```
GET /api/v1/voyages/recommend?start_port_id=X&end_port_id=Y&required_ton=Z

阶段1 - 航线过滤(O(N))：
  遍历所有 line_status=1 的航线
  解析 port_sequence → []int64
  过滤包含 startPortID 且 startIndex < endIndex 的航线

阶段2 - 航段计算(O(1))：
  计算 startIndex 到 endIndex 之间的所有相邻港对

阶段3 - 容量计算(O(M*K))：
  SELECT SUM(weight_ton) FROM segment_capacity_usage
  WHERE line_id=? AND vessel_id=? AND voyage_date=?
  AND segment_index BETWEEN ? AND ? GROUP BY segment_index
  瓶颈容量 = vessel.max_dwt - MAX(各航段已占用量)

阶段4 - 排序缓存(O(RlogR))：
  按瓶颈容量降序排列
  缓存 10 分钟（key: voyage:recommend:{start}:{end}:{ton}）
  订单创建/取消时清除缓存
```

### 7.3 订单状态机

```
允许的转换：
  0(待确认) → 1(已确认)  确认
  0(待确认) → 4(已取消)  取消
  1(已确认) → 2(运输中)  发货(记录实际离港时间+装货)
  1(已确认) → 4(已取消)  取消
  2(运输中) → 3(已完成)  完成(记录实际到港时间+卸货)
  2(运输中) → 4(已取消)  取消
  3(已完成)  — 终端状态
  4(已取消)  — 终端状态
```

### 7.4 航线生命周期

```
海运公司申请    → line_status=0(待审核)
管理员通过      → line_status=1(已启用)
管理员拒绝      → line_status=2(已弃用)

海运公司停用    → line_status=2(已弃用)
海运公司重新申请 → line_status=0(待审核)

管理员直接弃用  → line_status=2(已弃用)
管理员直接恢复  → line_status=1(已启用)
管理员永久删除  → 物理删除(delete_time)
```

---

## 8. 请求完整链路示例

### 创建订单

```
客户端 POST /api/v1/orders
  │
  ├── net/middleware/ (8个全局中间件)
  │   ├── Logger → Recovery → CORS → SecurityHeaders
  │   ├── Honeypot → IPBlocklist → RequestGuard → RateLimit
  │
  ├── net/middleware/auth.go
  │   └── RequireAuth() → JWT 解析 → c.Set("user_id", ..) / c.Set("role", ..)
  │
  ├── internal/handler/order.go
  │   └── CreateOrder()
  │       ├── c.ShouldBindJSON(&req)       → 解析请求体
  │       ├── validator.Validate(req)      → 参数校验
  │       ├── JWT 校验 shipper_company_id  → 越权检查
  │       └── h.svc.CreateOrder(ctx, req)  → 服务层
  │
  ├── internal/service/order_service.go
  │   ├── CostCalculator.Calculate
  │   ├── VesselDAO.GetByID
  │   ├── ShippingLineDAO.GetByID
  │   ├── PortSequenceParser.Parse
  │   ├── SegmentCalculator.Calculate
  │   ├── 查找/创建 cargo_note
  │   │
  │   └── db.Transaction(func(tx) error {
  │       ├── GET_LOCK("order_create", 10)
  │       ├── SELECT FOR UPDATE
  │       ├── CapacityChecker.Check
  │       ├── OrderNoGenerator.Generate
  │       ├── tx.Create(&order)
  │       ├── tx.Create(&cargos)
  │       ├── tx.Create(&usages)
  │       ├── AddCumulativeCapacity(tx, ...)
  │       └── RELEASE_LOCK
  │   })
  │
  └── response.Success(c.Writer, order)   → JSON 响应
```

### 更新订单状态（发货）

```
客户端 PUT /api/v1/orders/:id/status
  → 中间件链（同上）
  → Handler: UpdateOrderStatus()
     → 解析 status + port_id + actual_time + cargo_operations
     → Service: UpdateOrderStatus()
        → DAO: orderDAO.GetByID + 手动加载 LoadNote
        → 状态机: Transition(oldStatus, newStatus)
        → DAO: orderDAO.Update
        → 更新 voyage_berthing.actual_departure_time (status=2)
          或 voyage_berthing.actual_arrival_time (status=3)
        → 处理 cargo_operations: 查找/创建 voyage_cargo_note
        → WebSocket: PushOrderStatusUpdate
     → response.Success
```

### 港口访问记录

```
客户端 POST /api/v1/orders/:id/port-visit
  → 中间件链
  → Handler: RecordPortVisit()
     → 解析 port_id + actual_arrival + actual_departure + cargo_operations
     → Service: RecordPortVisit()
        → 校验订单 status == 2
        → 查找 voyage_berthing 获取 sequence_no
        → 更新 actual_arrival_time / actual_departure_time
        → 处理 cargo_operations
     → response.Success
```

---

## 9. API 接口总览

### 公开（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| GET | `/ws?token=` | WebSocket |
| POST | `/api/v1/auth/login` | 登录 |
| POST | `/api/v1/auth/refresh` | 刷新令牌 |
| POST | `/api/v1/shipper/register` | 货主注册 |
| POST | `/api/v1/shipping/register` | 船公司注册 |

### 需 JWT

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/auth/me` | 当前用户信息 |
| POST | `/api/v1/orders` | 创建订单 |
| GET | `/api/v1/orders` | 订单列表（keyword+status+sort） |
| GET | `/api/v1/orders/:id` | 订单详情 |
| POST | `/api/v1/orders/:id/cancel` | 取消订单 |
| PUT | `/api/v1/orders/:id/status` | 更新状态（含选港+时间+货物操作） |
| GET | `/api/v1/orders/:id/tracking` | 物流追踪 |
| POST | `/api/v1/orders/:id/pay` | 支付 |
| POST | `/api/v1/orders/:id/port-visit` | 港口访问记录 |
| GET | `/api/v1/voyages/recommend` | 航次推荐 |
| POST | `/api/v1/voyages/berthing` | 创建航次 |
| GET | `/api/v1/voyages/my` | 我的航次 |
| PUT | `/api/v1/berthings/:id/actual-times` | 更新靠泊时间 |
| GET | `/api/v1/cities` | 城市列表 |
| GET | `/api/v1/ports` | 港口列表 |
| GET | `/api/v1/ports/:id` | 港口详情 |
| GET | `/api/v1/vessels` | 船舶列表 |
| GET | `/api/v1/vessels/:id` | 船舶详情 |
| GET | `/api/v1/shipping-lines` | 航线列表 |
| GET | `/api/v1/shipping-lines/:id` | 航线详情 |
| GET | `/api/v1/shipping-lines/:id/port-sequence` | 港口序列 |
| DELETE | `/api/v1/shipping-lines/:id` | 删除航线（shipping 角色仅弃用） |
| POST | `/api/v1/shipping-lines/:id/reactivate` | 重新申请航线 |
| GET | `/api/v1/shipping-companies` | 船公司列表 |
| POST | `/api/v1/shipper/password/:id` | 货主改密 |
| POST | `/api/v1/shipping/password/:id` | 船公司改密 |
| GET | `/api/v1/export/ports` | 导出港口 |
| POST | `/api/v1/import/ports` | 导入港口 |
| GET | `/api/v1/export/vessels` | 导出船舶 |
| POST | `/api/v1/import/vessels` | 导入船舶 |
| GET | `/api/v1/export/shipping-lines` | 导出航线 |
| POST | `/api/v1/import/shipping-lines` | 导入航线 |
| GET | `/api/v1/export/orders` | 导出订单 |
| GET | `/api/v1/notifications` | 通知列表 |
| PUT | `/api/v1/notifications/:id/read` | 标记已读 |
| GET | `/api/v1/reports/orders` | 订单统计 |
| GET | `/api/v1/reports/voyage-utilization` | 航次利用率 |

### 需 admin

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/admin/list` | 管理员列表 |
| POST | `/api/v1/admin/register` | 创建管理员 |
| POST | `/api/v1/admin/password/:id` | 管理员改密 |
| GET | `/api/v1/admin/cargo/list` | 货物列表 |
| POST | `/api/v1/admin/cargo/create` | 新建货物 |
| DELETE | `/api/v1/admin/cargo/:id` | 删除货物 |
| GET | `/api/v1/admin/shipper/list` | 货主列表 |
| POST | `/api/v1/admin/shipper/:id/update` | 更新货主 |
| POST | `/api/v1/admin/shipper/:id/delete` | 删除货主 |
| GET | `/api/v1/admin/shipping/list` | 船公司列表 |
| POST | `/api/v1/admin/shipping/:id/update` | 更新船公司 |
| POST | `/api/v1/admin/shipping/:id/delete` | 删除船公司 |
| GET | `/api/v1/admin/shipping-lines/pending` | 待审核航线 |
| POST | `/api/v1/admin/shipping-lines/:id/approve` | 航线通过 |
| POST | `/api/v1/admin/shipping-lines/:id/deprecate` | 航线弃用 |
| POST | `/api/v1/admin/ports` | 创建港口 |
| PUT | `/api/v1/admin/ports/:id` | 更新港口 |
| DELETE | `/api/v1/admin/ports/:id` | 删除港口 |
| POST | `/api/v1/admin/vessels` | 创建船舶 |
| PUT | `/api/v1/admin/vessels/:id` | 更新船舶 |
| DELETE | `/api/v1/admin/vessels/:id` | 删除船舶 |
| POST | `/api/v1/admin/shipping-lines` | 创建航线 |
| PUT | `/api/v1/admin/shipping-lines/:id` | 更新航线 |
| DELETE | `/api/v1/admin/shipping-lines/:id` | 删除航线 |
| POST | `/api/v1/admin/cities` | 创建城市 |
| PUT | `/api/v1/admin/cities/:id` | 更新城市 |
| DELETE | `/api/v1/admin/cities/:id` | 删除城市 |
| POST | `/api/v1/admin/notifications` | 发送通知 |

---

## 10. 环境要求与启动

### 10.1 环境要求

- Go 1.21+
- MySQL 8.0+

### 10.2 启动步骤

```bash
# 1. 进入项目目录
cd backend

# 2. 安装 Go 依赖
go mod tidy

# 3. 创建数据库并执行建表脚本
mysql -u root -p < sql/tables_mysql.sql

# 4. 修改 config.yaml 中的数据库连接信息

# 5. 插入种子数据
go run ./cmd/seed

# 6. 启动服务
go run ./app

# 7. 验证启动成功
#    应看到：server started port=8080
```

### 10.3 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `DB_DSN` | MySQL 连接串 | config.yaml |
| `JWT_SECRET` | JWT 签名密钥 | config.yaml |
| `SERVER_PORT` | 监听端口 | 8080 |
| `LOG_LEVEL` | 日志级别 | debug |
| `LOG_OUTPUT_PATH` | 日志路径 | logs/app.log |
| `ENABLE_PPROF` | 启用 pprof | false |

---

## 11. 配置说明

详见 `config.yaml`：

```yaml
server:
  port: "8080"

database:
  driver: mysql
  dsn: "user:pass@tcp(127.0.0.1:3306)/mts?charset=utf8mb4&parseTime=True&loc=Local"
  max_open_conns: 25
  max_idle_conns: 10
  conn_max_lifetime: 1h

log:
  level: debug
  output_path: "logs/app.log"

jwt:
  secret: "your-production-secret-key"
  access_expire: 24h
  refresh_expire: 168h

cache:
  default_ttl: 10m

rate_limit:
  requests_per_second: 100
```

---

## 12. 测试账号

| 角色 | 用户名 | 密码 |
|------|--------|------|
| 管理员 | admin | admin123 |
| 货主01 | shipper01 | 123456 |
| 货主02 | shipper02 | 123456 |
| 货主03 | shipper03 | 123456 |
| 中远海运 | cosco | 123456 |
| 马士基 | maersk | 123456 |
| 地中海航运 | msc | 123456 |
| 达飞轮船 | cma | 123456 |

---

## 13. 常见问题

### Q1: 启动时端口被占用
```powershell
Get-Process -Name "app" -ErrorAction SilentlyContinue | Stop-Process -Force
```

### Q2: 登录返回 "invalid credentials"
运行种子脚本：`go run ./cmd/seed`

### Q3: 订单运费为 0
检查 `shipping_line` 表的字段是否完整。

### Q4: WebSocket 返回 401
路径是 `/ws`（不是 `/api/v1/ws`），且 token 未过期。

### Q5: 限流返回 429
默认 100/s，可调整 rate_limit 配置。

---

## 14. 性能与安全建议

1. **生产环境关闭 debug 模式**：`set GIN_MODE=release`
2. **JWT 密钥必须修改**：使用至少 32 字节随机字符串
3. **多实例部署需要改造**：go-cache → Redis，内存通知 → DB/Redis
4. **定期清理软删除记录**：超过 90 天的 delete_time 记录
5. **启用 pprof**：`set ENABLE_PPROF=true` → `/debug/pprof`
6. **数据库连接池调优**：根据实际负载调整 MaxOpenConns

---

*文档生成日期：2026-07-08*

