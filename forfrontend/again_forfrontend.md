# MTS 海运运输系统 — 角色权限与前端功能完整说明

> 本文档涵盖了：系统中有哪些角色、每个角色能做什么、每个功能的前后端交互流程、
> 后端做了什么、以及开发中需要注意的坑。

---

## 第一章：系统概览

### 1.1 这是什么系统

MTS（Maritime Transport System）是一个**海运运输管理平台**，解决的是"货主找船运货"和"船公司接单运输"的匹配问题。

**现实场景举例：**
- 一家贸易公司（货主）有 500 吨钢材要从上海运到鹿特丹。
- 这家公司登录系统，输入起止港口和货物吨数，系统推荐出可用的航次（由某条航线和某艘船舶组成的运输计划）。
- 货主选择一个航次，填写货物明细，系统自动计算运费，创建订单。
- 船公司登录系统，看到这个订单后确认运输，更新状态为"运输中"。
- 船公司完成运输后，将状态更新为"已完成"。
- 整个过程中，货主可以随时查看订单的物流追踪信息（靠泊时间、到港时间等）。

### 1.2 三种角色

系统中存在三种角色，他们的定位完全不同：

| 角色 | 身份 | 类比 | 典型页面 |
|------|------|------|----------|
| **shipper** | 货主（托运方） | 类似淘宝买家 | 创建订单、查订单、看物流 |
| **shipping** | 船公司（承运方） | 类似淘宝卖家 | 确认订单、更新物流状态 |
| **admin** | 系统管理员 | 类似平台运营 | 管理账号、发通知、删违规公司 |

**关键区别：** 角色决定了你能看到什么数据、能操作什么功能。
- shipper 只能看到**自己的**订单（user_id == shipper_company_id 强制校验）。
- shipping 可以看到**所有货主的**订单（方便统一调度）。
- admin 不能创建和操作订单，但可以**管理账号和发系统通知**。

### 1.3 前后端交互方式

```
前端（浏览器）                   后端（Go 服务）
    │                              │
    │─── POST /auth/login ────────→│  登录，获取 JWT 令牌
    │←── {access_token, ...} ─────│
    │                              │
    │─── GET /orders (带 Bearer) →│  请求受保护资源
    │←── {code:0, data: [...]} ───│
    │                              │
    │─── WebSocket /ws?token=... →│  建立实时推送连接
    │←── {type:"order_status_...} │  订单状态变更时自动推送
```

- **认证方式**：登录后获取 JWT token（access_token 有效期 15 分钟，refresh_token 有效期 7 天），后续所有请求在 Header 中带 `Authorization: Bearer <access_token>`。
- **令牌刷新**：access_token 过期后，用 refresh_token 调 `/auth/refresh` 获取新 access_token，不要要求用户重新登录。
- **响应格式**：所有 API 返回统一的 JSON 结构（`{code, message, data}`），`code=0` 表示成功，非零表示业务错误。

---

## 第二章：角色详解——货主（shipper）

### 2.1 是什么角色

货主是**有货物需要运输的公司或个人**。他们是系统的"需求方"——提交订单、支付运费、等待货物送达。

### 2.2 能做什么（完整功能列表）

#### 2.2.1 账号管理

**注册账号**
- 入口：公开接口，不需要登录即可访问。
- 需要填写：公司名称、登录用户名、密码（至少 6 位）。
- 后端做了什么：密码用 bcrypt 加密存储（不可逆加密，即使数据库泄露也无法还原密码）。
- 注册成功后自动返回公司信息，但**注册后不会自动登录**，前端应该跳转到登录页。

**登录**
- 三端共用同一个登录接口，通过 `role` 参数区分。
- 登录成功后返回：`access_token`（15分钟有效）、`refresh_token`（7天有效）、`role`、`user_id`。
- 前端需要保存这 4 个值。`user_id` 在后续创建订单时需要用到（作为 `shipper_company_id`）。

**修改密码**
- 需要提供旧密码验证。
- URL 中的 `{id}` 是当前用户的 company_id（即登录时返回的 user_id）。
- 为什么 URL 中要传 id：理论上可以从 JWT 中解析，但接口设计遵循 REST 风格，资源路径明确。

#### 2.2.2 订单操作（核心功能）

货主的全部分支操作都围绕"订单"展开。以下是典型操作流程：

```
浏览航线/船舶 → 查询航次推荐 → 创建订单 → 查看订单列表 → 查看物流追踪 → 取消订单(如需)
```

**① 浏览基础数据（前提）**

在下单之前，货主需要先了解有哪些航线、船舶、港口可用。这些数据由船公司通过 Excel 导入系统。

- `GET /shipping-lines` — 查看所有航线（名称、起止港、总里程）。
- `GET /vessels` — 查看所有船舶（名称、类型、载重吨 DWT）。
- `GET /ports` — 查看所有港口（名称、代码、所在城市）。
- `GET /ports/{id}` — 查看港口详情（含城市名、国家）。

港口数据有 10 分钟缓存——如果管理员修改了港口数据，前端最多 10 分钟才能看到更新。

**② 航次推荐（下单前的核心步骤）**

这是系统最核心的功能之一。货主输入三个参数：
- `start_port_id`：出发港口 ID（如 1 = 上海港）。
- `end_port_id`：目的港口 ID（如 3 = 鹿特丹港）。
- `required_ton`：需要运输的货物吨数（如 500 吨）。

后端做了什么：
1. 遍历系统所有航线，解析每条航线的港口序列（JSON 格式，如 `[1, 2, 3]`）。
2. 收集每条航线上有实际航次计划的（line_id + vessel_id + voyage_date 组合，来自 voyage_cargo_note 表）。
3. 对每个航次：计算从出发港到目的港之间所有"航段"的剩余容量，取**最小值**作为该航次的代表容量（瓶颈原则：整条航线的运力受限制于容量最小的航段）。
4. 筛选出代表容量 >= 需求吨数的航次，按剩余容量从大到小排序。
5. 结果缓存 1 分钟，1 分钟内相同参数直接从缓存返回。

**为什么叫"航段"？**

航线由多个港口顺序连接而成。例如航线"亚洲-欧洲快线"经过上海→新加坡→鹿特丹。从上海到鹿特丹的货物，经过两个航段：
- 航段 1：上海 → 新加坡
- 航段 2：新加坡 → 鹿特丹

每个航段上的运力是独立计算的（船舶在这段航线上有多少剩余空间）。如果航段 1 剩余 100 吨、航段 2 剩余 50 吨，那么整条航线最多只能运 50 吨（取最小值）。这就是"瓶颈原则"。

**③ 创建订单（核心操作）**

创建订单是系统中最复杂的操作——它涉及多个表的写入和并发控制。

请求参数（必填）：
| 参数 | 类型 | 说明 |
|------|------|------|
| shipper_company_id | Long | 货主 ID。**shipper 角色必须等于登录用户的 user_id**，否则返回 403 |
| city_id | Long | 货主所在城市 ID |
| line_id | Long | 航线 ID（从航次推荐结果中获取） |
| vessel_id | Long | 船舶 ID（从航次推荐结果中获取） |
| voyage_date | String | 航次日期，格式：`yyyy-MM-dd`（从航次推荐结果中获取） |
| start_port_id | Long | 出发港口 ID |
| end_port_id | Long | 目的港口 ID |
| cargo_items | Array | 货物列表（至少 1 项）。每项含名称、类型、重量(吨)、体积、单价等 |

请求参数（可选）：
| 参数 | 类型 | 说明 |
|------|------|------|
| shipper_contact | String | 发货方联系方式 |
| consignee_contact | String | 收货方联系方式 |
| expected_departure | String | 预计出发日期，格式：`yyyy-MM-dd` |
| expected_arrival | String | 预计到达日期，格式：`yyyy-MM-dd` |

**创建订单时后端执行的全部操作（按顺序）：**
1. **货物汇总**：计算所有货物的总重量、总体积、总金额。
2. **运费计算**：`总运费 = 总重量(吨) × 总航程(海里) × 基础费率(元/吨海里) × 货物系数`。基础费率和货物系数来自系统配置（config.yaml）。
3. **解析港口序列**：将航线的 JSON 港口序列解析为数组。
4. **计算航段**：根据起止港口计算途径的所有邻接航段。
5. **校验 cargo note**：检查出发港是否有 LOAD（装货）通知、目的港是否有 UNLOAD（卸货）通知——这是创建订单的前提条件。
6. **开启数据库事务**：
   a. **获取锁**：使用 MySQL `GET_LOCK` 获取该航次的互斥锁（锁名 = "voyage_{lineID}_{vesselID}_{date}"），防止两个货主同时下单导致超卖。
   b. **锁定航段**：`SELECT ... FOR UPDATE` 锁定涉及的所有航段容量记录。
   c. **容量检查**：每个航段上"已占用的吨位 + 新订单的吨位" <= 船舶最大载重。任一段超容则拒绝下单。
   d. **创建订单**：写入 shipping_order 表。
   e. **创建货物明细**：批量写入 order_cargo 表。
   f. **创建容量占用**：批量写入 segment_capacity_usage 表（每个航段一条记录）。
   g. **更新累积运力**：更新 voyage_cargo_note 的累积已预订容量。
   h. **释放锁**：`RELEASE_LOCK`。
7. **清除缓存**：删除所有以 `voyage_rec:` 开头的缓存（航次推荐结果），保证下次查询时能拿到最新的容量数据。
8. 返回创建的订单。

**为什么两种锁？**
- `GET_LOCK` 是互斥锁，确保同一航次同一时间只有一个订单在创建。
- `FOR UPDATE` 是行锁，确保当前事务读取的容量数据不被其他事务修改。
- 两者配合：先抢互斥锁（只有一个能进来），再用行锁精确锁定数据行。

**④ 查询订单列表**

- `GET /orders?shipper_company_id={id}&page=1&page_size=20&sort_by=create_time&sort_order=desc`
- shipper 角色：`{id}` 必须等于当前用户的 user_id，否则 403。
- 列表数据包含城市名称（通过 `Preload("City")` 联表查询）。
- 支持排序字段：create_time（默认）、order_no、total_weight_ton、order_status。

**⑤ 查询订单详情**

- 返回完整订单信息：含城市对象、货物明细数组、装货单/卸货单、出发港/目的港详情。
- 前端拿到数据后可以展示订单详情页。

**⑥ 物流追踪**

- 返回：订单状态、装卸货时间、起止港名称、计划/实际出发/到达时间、船舶名称、航线名称。
- 这个接口关联了 voyage_berthing（靠泊记录）表，提供实时的物流跟踪信息。

**⑦ 取消订单**

- 只能在订单状态为 0（草稿）、1（已确认）、2（运输中）时取消。
- 已完成的订单和已取消的订单不能再次取消。
- 取消后后端操作：软删除订单和货物明细、物理删除航段容量占用、释放累积运力、清除推荐缓存。

**⑧ 导出订单**

- `GET /export/orders?shipper_company_id={id}` 下载 Excel 文件。
- 文件名：orders.xlsx。浏览器直接触发下载。

**⑨ 实时接收通知**

- WebSocket 连接：`ws://localhost:8080/ws?token={access_token}`
- 当订单状态发生变更时（如船公司更新为"运输中"），后端向该订单的货主推送消息。
- 推送消息格式：
```json
{
  "type": "order_status_update",
  "order_id": 1,
  "status": 2,
  "timestamp": 1741234567
}
```
- **前端建议**：收到推送后，更新订单列表中对应订单的状态显示。不需要重新请求整个列表（除非需要最新数据）。连接断开后 3 秒自动重连。
- **为什么用 query 参数传 token 而不是 Header**：浏览器 `new WebSocket(url)` API 不支持自定义 Header。

### 2.3 典型页面

货主角色至少需要以下页面：

| 页面 | 路由建议 | 内容 |
|------|----------|------|
| 注册页 | /register | 公司名+用户名+密码，角色选 shipper |
| 登录页 | /login | 用户名+密码+角色选择 |
| 航次推荐 | /voyage-recommend | 三个输入框（起港/止港/吨数）+ 推荐结果列表 |
| 创建订单 | /order/create | 承接推荐结果，选航次+填货物明细 |
| 订单列表 | /orders | 分页表格，支持按状态筛选和排序 |
| 订单详情 | /orders/{id} | 完整信息展示 |
| 物流追踪 | /orders/{id}/tracking | 时间线或卡片形式展示 |
| 港口浏览 | /ports | 分页列表，点击弹出详情 |
| 船舶浏览 | /vessels | 分页列表 |
| 航线浏览 | /shipping-lines | 分页列表，含港口序列展示 |

---

## 第三章：角色详解——船公司（shipping）

### 3.1 是什么角色

船公司是**拥有船舶和航线的运输公司**。他们是系统的"供应方"——确认订单、安排运输、更新物流状态。

### 3.2 能做什么

#### 3.2.1 账号管理

**注册、登录、改密：**
- 与货主逻辑相同，只是注册接口是 `/shipping/register`，改密接口是 `/shipping/password/{id}`。
- 登录时 role 参数传 `shipping`。

#### 3.2.2 订单管理（核心功能）

船公司是实际操作订单流转的角色。流程如下：

```
查看待处理订单 → 确认运输 → 完成运输 → 查看报表
    ↓ (如需)
  取消订单
```

**① 查询订单**
- `GET /orders?shipper_company_id={id}&page=1&page_size=20`
- 与 shipper 不同：shipping 角色**不校验** shipper_company_id 是否匹配当前用户。也就是说船公司可以查看**任意货主**的订单——这是合理的，因为船公司需要统一调度所有订单。
- 列表同样包含城市名称。

**② 更新订单状态（核心操作）**

这是船公司最常用的功能。通过 `PUT /orders/{id}/status` 更新状态。

**订单状态转换规则（状态机）：**
```
  0(草稿) ───→ 1(已确认)     (货主创建订单后默认状态为 0)
  0(草稿) ───→ 4(已取消)     (货主或船公司取消)
  1(已确认) ─→ 2(运输中)     (船公司确认货物已装船)
  1(已确认) ─→ 4(已取消)     (取消)
  2(运输中) ─→ 3(已完成)     (船公司确认货物已送达)
  2(运输中) ─→ 4(已取消)     (取消，如运输途中出现问题)
  3(已完成) —/—→ 任何状态    (终态)
  4(已取消) —/—→ 任何状态    (终态)
```

**关键事件：** 每次状态变更后，后端的 WebSocket 服务会自动向该订单的货主推送状态更新消息。前端（shipper 端）收到推送后应该刷新订单状态显示。

**③ 查看基础数据**
- 与货主相同：可以查看港口、船舶、航线列表和详情。
- 这些数据通常是船公司自己通过 Excel 导入的，所以船公司可能更关注"我导入的数据是否正确"。

#### 3.2.3 数据管理（Excel 导入导出）

船公司负责维护基础数据（港口、船舶、航线）。这些数据通过 Excel 导入系统。

**导出（下载数据用于备份或编辑）：**
- `GET /export/ports` → 下载 ports.xlsx
- `GET /export/vessels` → 下载 vessels.xlsx
- `GET /export/shipping-lines` → 下载 shipping_lines.xlsx

**导入（上传编辑后的数据）：**
- `POST /import/ports` → 上传 .xlsx 文件，批量导入港口
- `POST /import/vessels` → 上传 .xlsx 文件，批量导入船舶
- `POST /import/shipping-lines` → 上传 .xlsx 文件，批量导入航线

**导入文件格式要求：**
- 第一行必须是表头（列名与导出格式一致）。
- 至少包含 1 行数据（不含表头）。
- 文件格式为 .xlsx（Office Open XML）。
- 上传方式：`multipart/form-data`，字段名 `file`。

**导出文件列说明：**

港口导出列：`ID, PortName, PortCode, CityID, Latitude, Longitude, PortType, MaxDraftMeter`

船舶导出列：`ID, VesselName, CallSign, IMO, VesselType, MaxDeadweightTon, GrossTonnage, NetTonnage, DraftMeter, SpeedKnot, ContainerTEU, IsAvailable, ShippingCompanyID`

航线导出列：`ID, LineName, ShippingCompanyID, PortSequence, TotalDistanceNm, DeparturePortName, DestinationPortName, Description`

订单导出列：`OrderID, OrderNo, ShipperCompanyID, CityID, LoadNoteID, UnloadNoteID, DeparturePortID, DestinationPortID, ExpectedDepartureDate, ExpectedArrivalDate, TotalCost, PaymentStatus, OrderStatus, TotalWeightTon, TotalVolumeCubicMeter, CreateTime`

#### 3.2.4 报表

船公司可以查看统计报表了解运营情况：

**订单统计报表：**
- `GET /reports/orders?start_date=2026-01-01&end_date=2026-07-06`
- 返回：总订单数、总重量（吨）、总体积（立方米）、总运费（元）、已完成数、已取消数、运输中数。
- 这些数据可以用于生成图表展示。

**航次利用率：**
- `GET /reports/voyage-utilization?line_id=1&vessel_id=1&voyage_date=2026-07-15`
- 返回：船舶最大载重（吨）、当前已占用（吨）、利用率（百分比）。
- 利用率 = 已占吨位 / 最大载重 × 100。例如最大 50000 吨、已占 500 吨，利用率为 1%。
- 这个指标帮助船公司了解运力使用效率。

### 3.3 典型页面

| 页面 | 路由建议 | 内容 |
|------|----------|------|
| 订单列表 | /shipping/orders | 可搜索任意货主订单，支持状态筛选 |
| 订单详情 | /shipping/orders/{id} | 完整信息 + 状态操作按钮 |
| 数据管理 | /data-management | 导入/导出港口、船舶、航线 |
| 报表 | /reports | 订单统计 + 航次利用率 |

---

## 第四章：角色详解——管理员（admin）

### 4.1 是什么角色

管理员是**系统运营方**。他们不参与具体的运输业务，而是负责管理平台上的账号和通信。

### 4.2 能做什么

#### 4.2.1 管理员账号管理

**创建管理员：**
- 一个管理员可以创建另一个管理员账号。
- 可以指定角色级别：1（超级管理员）、2（普通管理员），用于区分管理权限。

**修改管理员密码：**
- 需要提供旧密码验证。

#### 4.2.2 公司管理

**删除货主公司 / 删除船公司：**
- **软删除**——不是真的从数据库删除数据，而是设置 `delete_time` 字段。
- 删除后：该公司无法登录（查询时被 `WHERE delete_time IS NULL` 条件过滤）。
- 软删除的好处：可以恢复（把 delete_time 改回 NULL），数据不丢失。
- 删除接口返回 200 表示操作成功，与 HTTP DELETE 方法的语义不同（这里用的是 POST）。

#### 4.2.3 通知管理

**发送通知：**
- 向指定用户（user_id + role）发送通知。
- 通知类型：`order_created`（订单创建）、`order_cancelled`（订单取消）、`status_changed`（状态变更）。
- 如果 data 中提供 `email` 或 `phone` 字段，系统会异步发送真实邮件或短信（取决于通知提供商配置）。

**关于通知存储：**
- 通知存储在服务器**内存**中（`map[string][]Notification`）。
- 服务器重启后通知全部丢失——**不保证持久化**。
- 通知是"尽力投递"的实时消息，不是持久化的站内信。
- 如果需要持久化通知，后续可以改为数据库或 Redis 存储。

### 4.3 典型页面

| 页面 | 路由建议 | 内容 |
|------|----------|------|
| 管理后台 | /admin | 仪表盘，显示系统概览 |
| 管理员管理 | /admin/managers | 创建管理员、修改密码 |
| 发送通知 | /admin/notify | 选用户+写内容+发送 |
| 公司管理 | /admin/companies | 查看/删除货主和船公司 |

---

## 第五章：通用功能详解（所有角色共用）

### 5.1 登录页

**设计要点：**
- 用户名 + 密码 + 角色选择（下拉框或 tab 切换）。
- 登录成功后的处理：
  ```javascript
  // 保存令牌
  localStorage.setItem('access_token', data.access_token);
  localStorage.setItem('refresh_token', data.refresh_token);
  localStorage.setItem('user_id', data.user_id);
  localStorage.setItem('role', data.role);
  // 跳转到对应角色的首页
  if (data.role === 'shipper') router.push('/orders');
  if (data.role === 'shipping') router.push('/shipping/orders');
  if (data.role === 'admin') router.push('/admin');
  ```

### 5.2 Token 刷新机制

access_token 有效期只有 15 分钟。前端需要在 token 过期前自动刷新，不要让用户反复登录。

**推荐实现方案：**

```javascript
// 封装 fetch 或 axios 拦截器
async function request(url, options) {
  const token = localStorage.getItem('access_token');
  const res = await fetch(url, {
    ...options,
    headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' }
  });
  if (res.status === 401) {
    // token 过期，尝试刷新
    const refreshToken = localStorage.getItem('refresh_token');
    const refreshRes = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken })
    });
    if (refreshRes.ok) {
      const data = await refreshRes.json();
      localStorage.setItem('access_token', data.access_token);
      // 重试原始请求
      return request(url, options);
    } else {
      // 刷新失败，跳到登录页
      router.push('/login');
    }
  }
  return res;
}
```

**注意：** refresh_token 有效期为 7 天。如果 7 天内用户不活跃，refresh_token 也会过期，此时必须让用户重新登录。

### 5.3 WebSocket 连接

**连接时机：** 用户登录成功后立即建立 WebSocket 连接。
**断线重连：** 连接断开后等待 3 秒自动重连。
**消息处理：**

```javascript
const ws = new WebSocket(`ws://localhost:8080/ws?token=${accessToken}`);

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === 'order_status_update') {
    // 前端根据 msg.order_id 找到对应的订单项，更新其状态显示
    // 例如：把状态从 "已确认" 改为 "运输中"
    updateOrderStatus(msg.order_id, msg.status);
  }
};

ws.onclose = () => {
  setTimeout(() => {
    // 重连时需要使用最新的 access_token
    const token = localStorage.getItem('access_token');
    if (token) ws = new WebSocket(`ws://localhost:8080/ws?token=${token}`);
  }, 3000);
};
```

### 5.4 分页查询实现

所有列表接口（订单、港口、船舶、航线、通知）统一使用以下分页参数：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | Integer | 1 | 页码，从 1 开始 |
| page_size | Integer | 20 | 每页条数，最大 100 |

响应中的 `meta` 字段：
```json
{
  "code": 0,
  "message": "success",
  "data": [ ... ],
  "meta": {
    "page": 1,          // 当前页
    "page_size": 20,    // 每页条数
    "total": 150,       // 总记录数
    "total_pages": 8    // 总页数（服务端计算好的）
  }
}
```

注意：`total_pages` 是服务端计算好的，前端直接使用即可，不需要自己 `Math.ceil(total / page_size)`。

### 5.5 错误处理

**统一响应格式：**
```json
// 成功
{ "code": 0, "message": "success", "data": ... }

// 业务错误（code 非 0）
{ "code": 1003, "message": "order not found" }
```

**错误码说明：**

| code | HTTP 状态码 | 含义 | 前端处理 |
|------|------------|------|----------|
| 0 | 200 | 成功 | 正常处理 data |
| 1000 | 400 | 请求参数错误（必填字段缺失、格式错误） | 展示错误信息 |
| 1001 | 401 | 未认证（token 缺失或无效） | 跳转到登录页 |
| 1002 | 403 | 权限不足（角色不匹配、越权操作） | 提示"权限不足" |
| 1003 | 404 | 资源不存在 | 提示"未找到" |
| 1004 | 409 | 资源冲突（运力不足、订单已取消） | 展示具体冲突原因 |
| 1005 | 429 | 请求频率超限 | 提示"操作太频繁，请稍候" |
| 2000 | 500 | 服务器内部错误 | 提示"系统异常" |

**前端统一的错误处理建议：**
- 所有 API 请求都需要检查 `code` 字段，而不是 HTTP 状态码。
- 对于 401，立即跳转到登录页（清除本地存储的 token）。
- 对于 1000-1999，展示 message 给用户。
- 对于 2000+，展示"系统异常，请稍后重试"。

---

## 第六章：关键数据字段说明

### 6.1 订单状态

| 值 | 英文名 | 含义 | 谁可以触发 |
|----|--------|------|-----------|
| 0 | Draft | 草稿。订单刚创建，尚未生效 | 系统自动（创建时默认） |
| 1 | Confirmed | 已确认。货主确认下单 | 系统自动（创建时默认就是 1） |
| 2 | In Transit | 运输中。货物已装船出发 | shipping 角色更新 |
| 3 | Completed | 已完成。货物已送达 | shipping 角色更新 |
| 4 | Cancelled | 已取消。订单被取消 | shipper/shipping/admin 均可 |

**前端展示建议：** 用不同颜色的标签展示状态：
- 0/1：蓝色（待处理）
- 2：橙色（进行中）
- 3：绿色（已完成）
- 4：灰色（已取消）

### 6.2 城市与 city_id

- 创建订单时需要传 `city_id`（城市 ID）。
- 查询订单时，后端通过 `Preload("City")` 联表查询，返回 `city` 对象：`{ city_id: 1, city_name: "Shanghai" }`。
- **创建时只传 ID，名称是后端查询时返回的。** 前端不需要维护城市列表。

### 6.3 数据关系图（帮助前端理解数据结构）

```
shipping_order (订单)
  ├── shipper_company_id → shipper_company (货主公司)
  ├── city_id → city (城市)
  ├── departure_port_id → port (出发港)
  ├── destination_port_id → port (目的港)
  ├── load_note_id → voyage_cargo_note (装货单)
  ├── unload_note_id → voyage_cargo_note (卸货单)
  └── order_cargos[] → order_cargo (货物明细)
       ├── cargo_name (货物名称)
       ├── cargo_type (类型: bulk/container/liquid)
       ├── weight_ton (重量, 吨)
       ├── volume_cubic_meter (体积, 立方米)
       └── ...
```

---

## 第七章：开发注意事项（避坑指南）

### 7.1 常见误解

1. **"城市名称是传进来的"** → 不对。城市名称是后端联表查出来的，创建时只传 `city_id`。
2. **"通知是持久存储的"** → 不对。通知存在服务器内存中，重启后丢失。
3. **"取消订单是 DELETE 请求"** → 不对。取消订单是 `POST /orders/{id}/cancel`。
4. **"删除公司是 DELETE 请求"** → 不对。删除公司是 `POST /admin/shipper/{id}/delete`（软删除）。
5. **"WebSocket 路径是 /api/v1/ws"** → 不对。WebSocket 路径是 `/ws`。token 通过 query 参数传递。

### 7.2 权限校验总结

| 操作 | shipper | shipping | admin |
|------|---------|----------|-------|
| 创建订单 | 只能为自己创建 | 可为任意货主创建 | 可为任意货主创建 |
| 查看订单列表 | 只能看自己的 | 可看任意货主的 | 可看任意货主的 |
| 修改密码 | 只能改自己的 | 只能改自己的 | 可改任意 admin 的 |
| 删除公司 | 不可 | 不可 | 可以 |
| 发送通知 | 不可 | 不可 | 可以 |
| 更新订单状态 | 不可 | 可以 | 可以 |

### 7.3 接口路径中的 {id}

| 接口 | {id} 的含义 | 举例 |
|------|-------------|------|
| `/shipper/password/{id}` | 货主公司的 company_id | 1 |
| `/shipping/password/{id}` | 船公司的 company_id | 1 |
| `/orders/{id}` | 订单的 order_id | 1 |
| `/ports/{id}` | 港口的 port_id | 1 |
| `/vessels/{id}` | 船舶的 vessel_id | 1 |
| `/shipping-lines/{id}` | 航线的 line_id | 1 |
| `/notifications/{id}` | 通知的 id（字符串） | "notif_1741234567890" |
| `/admin/password/{id}` | 管理员的 admin_id | 1 |
| `/admin/shipper/{id}/delete` | 货主公司的 company_id | 1 |
| `/admin/shipping/{id}/delete` | 船公司的 company_id | 1 |

### 7.4 Excel 导入注意事项

- 文件格式必须是 `.xlsx`（不是 `.xls`，也不是 `.csv`）。
- 第一行必须是表头。
- 上传方式：`multipart/form-data`，字段名固定为 `file`。
- 成功导入后返回 `{"code":0, "data": {"imported": 10}}`，表示成功导入 10 条记录。
- 如果某行数据有问题，不会部分导入——要么全部成功，要么遇到第一个错误时整体失败并报错。

---

## 附录：API 路径速查总表

### 公开接口（无需 Token）

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | /health | 健康检查，返回 {"status":"ok"} |
| GET | /ws?token=xxx | WebSocket 实时推送连接 |
| POST | /api/v1/auth/login | 用户登录 |
| POST | /api/v1/auth/refresh | 刷新 access_token |
| POST | /api/v1/shipper/register | 货主注册 |
| POST | /api/v1/shipping/register | 船公司注册 |

### 受保护接口（需 Bearer Token）

| 方法 | 路径 | 功能 | 可用角色 |
|------|------|------|----------|
| POST | /api/v1/shipper/password/{id} | 货主改密 | shipper |
| POST | /api/v1/shipping/password/{id} | 船公司改密 | shipping |
| POST | /api/v1/orders | 创建订单 | shipper/shipping/admin |
| GET | /api/v1/orders/{id} | 订单详情 | shipper/shipping/admin |
| POST | /api/v1/orders/{id}/cancel | 取消订单 | shipper/shipping/admin |
| PUT | /api/v1/orders/{id}/status | 更新状态 | shipping/admin |
| GET | /api/v1/orders | 订单列表 | shipper/shipping/admin |
| GET | /api/v1/orders/{id}/tracking | 物流追踪 | shipper/shipping/admin |
| GET | /api/v1/voyages/recommend | 航次推荐 | 全部 |
| GET | /api/v1/ports | 港口列表 | 全部 |
| GET | /api/v1/ports/{id} | 港口详情 | 全部 |
| GET | /api/v1/vessels | 船舶列表 | 全部 |
| GET | /api/v1/vessels/{id} | 船舶详情 | 全部 |
| GET | /api/v1/shipping-lines | 航线列表 | 全部 |
| GET | /api/v1/shipping-lines/{id} | 航线详情 | 全部 |
| GET | /api/v1/shipping-lines/{id}/port-sequence | 港口序列 | 全部 |
| GET | /api/v1/export/ports | 导出港口 | 全部 |
| POST | /api/v1/import/ports | 导入港口 | 全部 |
| GET | /api/v1/export/vessels | 导出船舶 | 全部 |
| POST | /api/v1/import/vessels | 导入船舶 | 全部 |
| GET | /api/v1/export/shipping-lines | 导出航线 | 全部 |
| POST | /api/v1/import/shipping-lines | 导入航线 | 全部 |
| GET | /api/v1/export/orders?shipper_company_id= | 导出订单 | 全部 |
| GET | /api/v1/notifications | 通知列表 | 全部 |
| PUT | /api/v1/notifications/{id}/read | 标记已读 | 全部 |
| GET | /api/v1/reports/orders | 订单统计 | 全部 |
| GET | /api/v1/reports/voyage-utilization | 航次利用率 | 全部 |

### 管理员专用（需 Bearer Token + role=admin）

| 方法 | 路径 | 功能 |
|------|------|------|
| POST | /api/v1/admin/register | 创建管理员 |
| POST | /api/v1/admin/password/{id} | 修改管理员密码 |
| POST | /api/v1/admin/notifications | 发送通知 |
| POST | /api/v1/admin/shipper/{id}/delete | 删除货主公司 |
| POST | /api/v1/admin/shipping/{id}/delete | 删除船公司 |
