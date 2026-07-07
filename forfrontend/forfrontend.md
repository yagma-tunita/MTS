# MTS 海运运输系统 — 前端接口字典

> **基础地址**: `http://localhost:8080/api/v1`  
> **数据格式**: JSON  
> **编码**: UTF-8  
> **Swagger 在线文档**: 启动后访问 `http://localhost:8080/swagger/index.html`

---

## 目录

- [统一规范](#统一规范)
- [1. 登录](#1-登录)
- [2. 刷新 Token](#2-刷新-token)
- [3. 货主注册](#3-货主注册)
- [4. 船公司注册](#4-船公司注册)
- [5. 货主修改密码](#5-货主修改密码)
- [6. 船公司修改密码](#6-船公司修改密码)
- [7. 创建订单](#7-创建订单)
- [8. 查询单个订单](#8-查询单个订单)
- [9. 查询订单列表](#9-查询订单列表)
- [10. 更新订单状态](#10-更新订单状态)
- [11. 取消订单](#11-取消订单)
- [12. 订单跟踪](#12-订单跟踪)
- [13. 航次推荐](#13-航次推荐)
- [14. 查询港口列表](#14-查询港口列表)
- [15. 查询单个港口](#15-查询单个港口)
- [16. 查询船舶列表](#16-查询船舶列表)
- [17. 查询单个船舶](#17-查询单个船舶)
- [18. 查询航线列表](#18-查询航线列表)
- [19. 查询单个航线](#19-查询单个航线)
- [20. 查询航线港口序列](#20-查询航线港口序列)
- [21. 查询通知列表](#21-查询通知列表)
- [22. 标记通知已读](#22-标记通知已读)
- [23. 管理员创建管理员](#23-管理员创建管理员)
- [24. 管理员修改密码](#24-管理员修改密码)
- [25. 管理员发送通知](#25-管理员发送通知)
- [26. 订单统计报表](#26-订单统计报表)
- [27. 航次利用率报表](#27-航次利用率报表)
- [28. Excel 导出](#28-excel-导出)
- [29. Excel 导入](#29-excel-导入)
- [30. WebSocket](#30-websocket)
- [31. 健康检查](#31-健康检查)
- [32. 管理员删除货主公司](#32-管理员删除货主公司)
- [33. 管理员删除船运公司](#33-管理员删除船运公司)
- [34. 管理员查询货主公司列表](#34-管理员查询货主公司列表)
- [35. 管理员查询船公司列表](#35-管理员查询船公司列表)
- [36. 管理员查询管理员列表](#36-管理员查询管理员列表)
- [37. 管理员更新货主公司](#37-管理员更新货主公司)
- [38. 管理员更新船公司](#38-管理员更新船公司)
- [39. 更新靠泊实际时间](#39-更新靠泊实际时间)
- [40. 货主查看船公司列表](#40-货主查看船公司列表)
- [41. 查询城市列表](#41-查询城市列表)
- [42. 虚拟支付](#42-虚拟支付)
- [43. 航线申请（创建航次靠泊记录）](#43-航线申请创建航次靠泊记录)
- [44. 管理员查询货物列表](#44-管理员查询货物列表)
- [45. 管理员创建港口](#45-管理员创建港口)
- [46. 管理员更新港口](#46-管理员更新港口)
- [47. 管理员删除港口](#47-管理员删除港口)
- [48. 管理员创建船舶](#48-管理员创建船舶)
- [49. 管理员更新船舶](#49-管理员更新船舶)
- [50. 管理员删除船舶](#50-管理员删除船舶)
- [51. 管理员创建航线](#51-管理员创建航线)
- [52. 管理员更新航线](#52-管理员更新航线)
- [53. 管理员删除航线](#53-管理员删除航线)

---

## 统一规范

### 响应格式

**成功（无分页）：**
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

**成功（有分页）：**
```json
{
  "code": 0,
  "message": "success",
  "data": [ ... ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

**失败（所有接口通用）：**
```json
{
  "code": 1003,
  "message": "order not found"
}
```

### 错误码

| code | HTTP 状态码 | 含义 |
|------|------------|------|
| 0 | 200 | 成功 |
| 1000 | 400 | 请求参数错误 |
| 1001 | 401 | 未授权（token 无效或过期） |
| 1002 | 403 | 权限不足 |
| 1003 | 404 | 资源未找到 |
| 1004 | 409 | 资源冲突（如运力不足） |
| 1005 | 429 | 请求过于频繁 |
| 2000 | 500 | 服务器内部错误 |

### 认证方式

受保护接口统一在请求头携带：
```
Authorization: Bearer <access_token>
Content-Type: application/json
```

---

## 1. 登录

> `POST /auth/login` — 公开接口，无需登录

**功能说明：** 用户登录，三种角色（shipper/shipping/admin）用用户名+密码登录，返回 JWT 双令牌（access_token 有效期 15 分钟，refresh_token 有效期 7 天）

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | String | 1 | 用户名 |
| password | String | 1 | 密码 |
| role | String | 1 | 角色：`shipper`(货主) / `shipping`(船公司) / `admin`(管理员) |

### 请求示例

```json
{
  "username": "test001",
  "password": "123456",
  "role": "shipper"
}
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "role": "shipper",
    "user_id": 1
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| access_token | String | 访问令牌，有效期 15 分钟 |
| refresh_token | String | 刷新令牌，有效期 7 天，用于获取新的 access_token |
| role | String | 用户角色：shipper(货主) / shipping(船公司) / admin(管理员) |
| user_id | Integer | 用户（公司/管理员）ID |

**token 说明：** access_token 有效期 15 分钟，refresh_token 有效期 7 天。

---

## 2. 刷新 Token

> `POST /auth/refresh` — 公开接口

**功能说明：** 用登录时返回的 refresh_token 换取新的 access_token，无需重新输入密码。refresh_token 有效期 7 天

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| refresh_token | String | 1 | 登录时返回的 refresh_token |

### 请求示例

```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs..."
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| access_token | String | 新的访问令牌，有效期 15 分钟 |

---

## 3. 货主注册

> `POST /shipper/register` — 公开接口

**功能说明：** 创建货主公司账号，填写公司名称+登录用户名+密码。密码 bcrypt 加密存储。用户名全局唯一

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| company_name | String | 1 | 公司名称 |
| login_username | String | 1 | 登录用户名（全局唯一） |
| password | String | 1 | 密码，至少 6 位 |
| unified_social_credit_code | String | 0 | 统一社会信用代码 |
| legal_representative | String | 0 | 法定代表人 |
| contact_phone | String | 0 | 联系电话 |
| address | String | 0 | 公司地址 |

### 请求示例

```json
{
  "company_name": "Global Trade Co.",
  "login_username": "test001",
  "password": "123456",
  "unified_social_credit_code": "91440101MA5XXXXXXX",
  "legal_representative": "张三",
  "contact_phone": "13800138000",
  "address": "上海市浦东新区陆家嘴金融中心"
}
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "company_id": 1,
    "company_name": "Global Trade Co.",
    "unified_social_credit_code": "91440101MA5XXXXXXX",
    "legal_representative": "张三",
    "contact_phone": "13800138000",
    "address": "上海市浦东新区陆家嘴金融中心",
    "login_username": "test001",
    "account_status": 1,
    "create_time": "2026-07-03T12:00:00Z",
    "update_time": "2026-07-03T12:00:00Z",
    "delete_time": null
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| company_id | Integer | 货主公司 ID |
| company_name | String | 公司名称 |
| unified_social_credit_code | String | 统一社会信用代码（可为 null） |
| legal_representative | String | 法定代表人（可为 null） |
| contact_phone | String | 联系电话（可为 null） |
| address | String | 公司地址（可为 null） |
| login_username | String | 登录用户名 |
| account_status | Integer | 账户状态：1=正常，0=禁用 |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |
| delete_time | String | 删除时间（软删除标记，null 表示未删除） |

---

## 4. 船公司注册

> `POST /shipping/register` — 公开接口

**功能说明：** 创建船运公司账号，填写公司名称+登录用户名+密码。密码 bcrypt 加密存储。用户名全局唯一

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| company_name | String | 1 | 公司名称 |
| login_username | String | 1 | 登录用户名 |
| password | String | 1 | 密码，至少 6 位 |
| unified_social_credit_code | String | 0 | 统一社会信用代码 |
| contact_person | String | 0 | 联系人 |
| contact_phone | String | 0 | 联系电话 |
| address | String | 0 | 公司地址 |

### 请求示例

```json
{
  "company_name": "Oceanic Shipping Co.",
  "login_username": "shipping001",
  "password": "123456",
  "unified_social_credit_code": "SHIP20240001",
  "contact_person": "John Smith",
  "contact_phone": "+65-12345678",
  "address": "12 Harbor Road, Singapore"
}
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "company_id": 1,
    "company_name": "Oceanic Shipping Co.",
    "unified_social_credit_code": "SHIP20240001",
    "contact_person": "John Smith",
    "contact_phone": "+65-12345678",
    "address": "12 Harbor Road, Singapore",
    "login_username": "shipping001",
    "account_status": 1,
    "create_time": "2026-07-03T12:00:00Z",
    "update_time": "2026-07-03T12:00:00Z",
    "delete_time": null
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| company_id | Integer | 船公司 ID |
| company_name | String | 公司名称 |
| unified_social_credit_code | String | 统一社会信用代码（可为 null） |
| contact_person | String | 联系人（可为 null） |
| contact_phone | String | 联系电话（可为 null） |
| address | String | 公司地址（可为 null） |
| login_username | String | 登录用户名 |
| account_status | Integer | 账户状态：1=正常，0=禁用 |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |
| delete_time | String | 删除时间（软删除标记，null 表示未删除） |

---

## 5. 货主修改密码

> `POST /shipper/password/{id}` — 需 Bearer Token  
> 注意：`{id}` 必须是当前登录用户的 company_id，否则 403

**功能说明：** 货主修改自己的登录密码。需验证旧密码正确，新密码 bcrypt 加密后更新。{id} 必须等于 JWT 中的 user_id（shipper 角色时强制校验）

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| old_password | String | 1 | 旧密码 |
| new_password | String | 1 | 新密码，至少 6 位 |

### 请求示例

```json
{
  "old_password": "123456",
  "new_password": "654321"
}
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "password updated"
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| message | String | 操作结果提示信息 |

---

## 6. 船公司修改密码

> `POST /shipping/password/{id}` — 需 Bearer Token

**功能说明：** 船公司修改自己的登录密码。需验证旧密码正确，新密码 bcrypt 加密后更新

请求参数、请求示例、响应示例、返回数据字段同第 5 节。

---

## 7. 创建订单

> `POST /orders` — 需 Bearer Token

**功能说明：** 货主选择航次/船舶/起止港后提交订单。支持多货物。系统自动计算运费（总重量×总海里×费率×货类系数）、校验船舶各航段剩余运力（GET_LOCK 防并发超卖）、写入订单+货物+运力占用多表事务。创建成功后清除航次推荐缓存

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| shipper_company_id | Long | 1 | 货主 ID，必须等于 JWT 中的 user_id |
| city_id | Long | 1 | 城市 ID |
| line_id | Long | 1 | 航线 ID |
| vessel_id | Long | 1 | 船舶 ID |
| voyage_date | String | 1 | 航次日期，格式 yyyy-MM-dd |
| start_port_id | Long | 1 | 出发港口 ID |
| end_port_id | Long | 1 | 目的港口 ID |
| cargo_items | CargoItem[] | 1 | 货物列表，至少 1 项 |
| shipper_contact | String | 0 | 发货方联系方式 |
| consignee_contact | String | 0 | 收货方联系方式 |
| expected_departure | String | 0 | 预计出发日期 yyyy-MM-dd |
| expected_arrival | String | 0 | 预计到达日期 yyyy-MM-dd |

**CargoItem 结构：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| cargo_name | String | 1 | 货物名称 |
| cargo_type | String | 1 | 类型：bulk(散货) / container(集装箱) / liquid(液体) |
| quantity | Double | 1 | 数量 |
| weight_ton | Double | 1 | 重量（吨），用于运力计算和运费 |
| volume_cub_m | Double | 1 | 体积（立方米） |
| unit_price | Double | 1 | 单价 |
| subtotal | Double | 1 | 小计 |

### 请求示例

```json
{
  "shipper_company_id": 1,
  "city_id": 1,
  "line_id": 1,
  "vessel_id": 1,
  "voyage_date": "2026-07-15",
  "start_port_id": 1,
  "end_port_id": 3,
  "cargo_items": [
    {
      "cargo_name": "钢材",
      "cargo_type": "bulk",
      "quantity": 100,
      "weight_ton": 500,
      "volume_cub_m": 200,
      "unit_price": 50,
      "subtotal": 5000
    }
  ],
  "shipper_contact": "张三-13800138000",
  "consignee_contact": "李四-13900139000",
  "expected_departure": "2026-07-10",
  "expected_arrival": "2026-07-20"
}
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "order_id": 1,
    "order_no": "ORD20260703abc12345",
    "shipper_company_id": 1,
    "city_id": 1,
    "load_note_id": 1,
    "unload_note_id": 2,
    "departure_port_id": 1,
    "destination_port_id": 3,
    "expected_departure_date": "2026-07-10",
    "expected_arrival_date": "2026-07-20",
    "total_cost": 212500,
    "shipper_contact": "张三-13800138000",
    "consignee_contact": "李四-13900139000",
    "payment_status": 0,
    "order_status": 1,
    "total_weight_ton": 500,
    "total_volume_cubic_meter": 200,
    "create_time": "2026-07-03T12:00:00Z",
    "update_time": "2026-07-03T12:00:00Z",
    "delete_time": null,
    "order_cargos": [
      {
        "detail_id": 1,
        "order_id": 1,
        "cargo_name": "钢材",
        "cargo_type": "bulk",
        "quantity": 100,
        "weight_ton": 500,
        "volume_cubic_meter": 200,
        "unit_price": 50,
        "subtotal": 5000,
        "create_time": "2026-07-03T12:00:00Z",
        "update_time": "2026-07-03T12:00:00Z",
        "delete_time": null
      }
    ]
  }
}
```

### 返回数据字段

**订单主表字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| order_id | Integer | 订单 ID |
| order_no | String | 订单编号（全局唯一） |
| shipper_company_id | Integer | 货主公司 ID |
| city_id | Integer | 城市 ID |
| load_note_id | Integer | 装货通知单 ID |
| unload_note_id | Integer | 卸货通知单 ID |
| departure_port_id | Integer | 出发港口 ID |
| destination_port_id | Integer | 目的港口 ID |
| expected_departure_date | String | 预计出发日期 (yyyy-MM-dd) |
| expected_arrival_date | String | 预计到达日期 (yyyy-MM-dd) |
| total_cost | Double | 总运费 |
| shipper_contact | String | 发货方联系方式 |
| consignee_contact | String | 收货方联系方式 |
| payment_status | Integer | 支付状态：0=未支付 |
| order_status | Integer | 订单状态：0=草稿, 1=已确认, 2=运输中, 3=已完成, 4=已取消 |
| total_weight_ton | Double | 总重量（吨） |
| total_volume_cubic_meter | Double | 总体积（立方米） |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |
| delete_time | String | 删除时间（null 表示未删除） |

**order_cargos（货物明细）子字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| detail_id | Integer | 货物明细 ID |
| order_id | Integer | 关联订单 ID |
| cargo_name | String | 货物名称 |
| cargo_type | String | 货物类型：bulk(散货) / container(集装箱) / liquid(液体) |
| quantity | Double | 数量 |
| weight_ton | Double | 重量（吨） |
| volume_cubic_meter | Double | 体积（立方米） |
| unit_price | Double | 单价 |
| subtotal | Double | 小计金额 |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |
| delete_time | String | 删除时间（null 表示未删除） |

### 订单状态

| order_status | 含义 |
|-------------|------|
| 0 | 草稿 (Draft) |
| 1 | 已确认 (Confirmed) |
| 2 | 运输中 (In Transit) |
| 3 | 已完成 (Completed) |
| 4 | 已取消 (Cancelled) |

payment_status：0 = 未支付

---

## 8. 查询单个订单

> `GET /orders/{id}` — 需 Bearer Token

**功能说明：** 根据订单 ID 查询完整订单信息，包括订单主数据、货物明细列表、关联的装货单/卸货单、起运港/目的港详情、货主公司信息

### 请求参数

无请求体，{id} 为路径参数（订单 ID）。

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "order_id": 1,
    "order_no": "ORD20260703abc12345",
    "shipper_company_id": 1,
    "city_id": 1,
    "load_note_id": 1,
    "unload_note_id": 2,
    "departure_port_id": 1,
    "destination_port_id": 3,
    "total_cost": 212500,
    "order_status": 1,
    "payment_status": 0,
    "total_weight_ton": 500,
    "total_volume_cubic_meter": 200,
    "shipper_contact": "张三-13800138000",
    "consignee_contact": "李四-13900139000",
    "create_time": "2026-07-03T12:00:00Z",
    "update_time": "2026-07-03T12:00:00Z",
    "shipper_company": {
      "company_id": 1,
      "company_name": "Global Trade Co.",
      "login_username": "test001"
    },
    "city": {
      "city_id": 1,
      "city_name": "Shanghai"
    },
    "load_note": {
      "note_id": 1,
      "operation_type": "LOAD",
      "cargo_name": "钢材"
    },
    "unload_note": {
      "note_id": 2,
      "operation_type": "UNLOAD",
      "cargo_name": "钢材"
    },
    "departure_port": {
      "port_id": 1,
      "port_name": "Shanghai Port"
    },
    "destination_port": {
      "port_id": 3,
      "port_name": "Rotterdam Port"
    },
    "order_cargos": [
      {
        "detail_id": 1,
        "cargo_name": "钢材",
        "cargo_type": "bulk",
        "quantity": 100,
        "weight_ton": 500,
        "volume_cubic_meter": 200,
        "unit_price": 50,
        "subtotal": 5000
      }
    ]
  }
}
```

### 返回数据字段

**订单主表字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| order_id | Integer | 订单 ID |
| order_no | String | 订单编号 |
| shipper_company_id | Integer | 货主公司 ID |
| city_id | Integer | 城市 ID |
| load_note_id | Integer | 装货通知单 ID |
| unload_note_id | Integer | 卸货通知单 ID |
| departure_port_id | Integer | 出发港口 ID |
| destination_port_id | Integer | 目的港口 ID |
| total_cost | Double | 总运费 |
| order_status | Integer | 订单状态：0=草稿, 1=已确认, 2=运输中, 3=已完成, 4=已取消 |
| payment_status | Integer | 支付状态：0=未支付 |
| total_weight_ton | Double | 总重量（吨） |
| total_volume_cubic_meter | Double | 总体积（立方米） |
| shipper_contact | String | 发货方联系方式 |
| consignee_contact | String | 收货方联系方式 |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |

**嵌套对象字段：**

| 对象 | 字段 | 类型 | 说明 |
|------|------|------|------|
| shipper_company | company_id | Integer | 货主公司 ID |
| shipper_company | company_name | String | 货主公司名称 |
| shipper_company | login_username | String | 货主登录用户名 |
| city | city_id | Integer | 城市 ID |
| city | city_name | String | 城市名称 |
| load_note | note_id | Integer | 装货通知单 ID |
| load_note | operation_type | String | 操作类型：LOAD(装货) |
| load_note | cargo_name | String | 货物名称 |
| unload_note | note_id | Integer | 卸货通知单 ID |
| unload_note | operation_type | String | 操作类型：UNLOAD(卸货) |
| unload_note | cargo_name | String | 货物名称 |
| departure_port | port_id | Integer | 出发港口 ID |
| departure_port | port_name | String | 出发港口名称 |
| destination_port | port_id | Integer | 目的港口 ID |
| destination_port | port_name | String | 目的港口名称 |

**order_cargos（货物明细）子字段同第 7 节。**

---

## 9. 查询订单列表

> `GET /orders?shipper_company_id=1&page=1&page_size=20&sort_by=create_time&sort_order=desc` — 需 Bearer Token

**功能说明：** 按货主公司 ID 分页查询订单列表，支持按创建时间/订单号/总重量/状态排序。shipper 角色只能查自己的订单（JWT 校验 user_id），admin/shipping 角色可查任意货主

### 请求参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| shipper_company_id | Long | 1 | | 货主 ID |
| page | Integer | 0 | 1 | 页码 |
| page_size | Integer | 0 | 20 | 每页条数（最大 100） |
| sort_by | String | 0 | create_time | 排序字段：create_time / order_no / total_weight_ton / order_status |
| sort_order | String | 0 | desc | asc 或 desc |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "order_id": 1,
      "order_no": "ORD20260703abc12345",
      "shipper_company_id": 1,
      "order_status": 1,
      "total_weight_ton": 500,
      "total_cost": 212500,
      "create_time": "2026-07-03T12:00:00Z",
      "city": {
        "city_id": 1,
        "city_name": "Shanghai"
      }
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 50,
    "total_pages": 3
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| order_id | Integer | 订单 ID |
| order_no | String | 订单编号 |
| shipper_company_id | Integer | 货主公司 ID |
| order_status | Integer | 订单状态 |
| total_weight_ton | Double | 总重量（吨） |
| total_cost | Double | 总运费 |
| create_time | String | 创建时间 (ISO 8601) |

**嵌套对象字段：**

| 对象 | 字段 | 类型 | 说明 |
|------|------|------|------|
| city | city_id | Integer | 城市 ID |
| city | city_name | String | 城市名称 |

**meta（分页信息）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| page | Integer | 当前页码 |
| page_size | Integer | 每页条数 |
| total | Integer | 总记录数 |
| total_pages | Integer | 总页数 |

---

## 10. 更新订单状态

> `PUT /orders/{id}/status` — 需 Bearer Token

**功能说明：** 按状态机规则更新订单状态（0草稿→1已确认→2运输中→3已完成，任何状态→4已取消）。状态变更后通过 WebSocket 向该订单货主实时推送状态更新消息

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| status | Integer | 1 | 目标状态：0~4 |

### 请求示例

```json
{
  "status": 2
}
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "order status updated"
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| message | String | 操作结果提示信息 |

### 状态转换规则

0(草稿) -> 1(已确认)  
0(草稿) -> 4(已取消)  
1(已确认) -> 2(运输中)  
1(已确认) -> 4(已取消)  
2(运输中) -> 3(已完成)  
2(运输中) -> 4(已取消)  
3(已完成) / 4(已取消) -> 终态，不可转换

WebSocket 推送：更新后会向该订单的货主推送状态变更消息。

---

## 11. 取消订单

> `POST /orders/{id}/cancel` — 需 Bearer Token

**功能说明：** 取消指定订单。后端操作：软删除订单及货物明细，释放航段运力占用（物理删除 segment_capacity_usage），更新装/卸货单累计吨位，清除航次推荐缓存

### 请求参数

无请求体。

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "order cancelled"
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| message | String | 操作结果提示信息 |

后端操作：软删除订单及货物明细，删除段容量占用，释放占用的累计运力。

---

## 12. 订单跟踪

> `GET /orders/{id}/tracking` — 需 Bearer Token

**功能说明：** 查询订单的完整物流跟踪信息，包括装货时间、卸货时间、起运港/目的港的靠泊计划时间和实际出发/到达时间、船舶名称、航线名称等

### 请求参数

无请求体，{id} 为订单 ID。

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "order_id": 1,
    "order_no": "ORD20260703abc12345",
    "order_status": 2,
    "status_name": "In Transit",
    "load_time": "2026-07-15T08:00:00Z",
    "unload_time": null,
    "departure_port": "Shanghai Port",
    "destination_port": "Rotterdam Port",
    "expected_departure": "2026-07-10T00:00:00+08:00",
    "expected_arrival": "2026-07-20T00:00:00+08:00",
    "departure_berthing_id": 1,
    "arrival_berthing_id": 3,
    "departure_planned": "2026-07-15T08:00:00Z",
    "departure_actual": null,
    "arrival_planned": "2026-07-22T10:00:00Z",
    "arrival_actual": null,
    "vessel_name": "M/V Ocean Queen",
    "line_name": "Asia-Europe Express"
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| order_id | Integer | 订单 ID |
| order_no | String | 订单编号 |
| order_status | Integer | 订单状态：0=草稿, 1=已确认, 2=运输中, 3=已完成, 4=已取消 |
| status_name | String | 状态英文名称，如 "In Transit" |
| load_time | String | 装货时间 (ISO 8601，可为 null) |
| unload_time | String | 卸货时间 (ISO 8601，可为 null) |
| departure_port | String | 出发港口名称 |
| destination_port | String | 目的港口名称 |
| expected_departure | String | 客户预计出发时间 (ISO 8601，可为 null，创建订单时填写) |
| expected_arrival | String | 客户预计到达时间 (ISO 8601，可为 null，创建订单时填写) |
| departure_berthing_id | Integer | 出发港靠泊记录 ID，用于 `PUT /berthings/{id}/actual-times` 更新实际时间 |
| arrival_berthing_id | Integer | 目的港靠泊记录 ID，同上 |
| departure_planned | String | 计划出发时间 (ISO 8601) |
| departure_actual | String | 实际出发时间 (ISO 8601，可为 null) |
| arrival_planned | String | 计划到达时间 (ISO 8601) |
| arrival_actual | String | 实际到达时间 (ISO 8601，可为 null) |
| vessel_name | String | 船舶名称 |
| line_name | String | 航线名称 |

状态名映射：0=Draft, 1=Confirmed, 2=In Transit, 3=Completed, 4=Cancelled

> **更新实际时间：** `departure_actual` 和 `arrival_actual` 字段通过 `PUT /berthings/{id}/actual-times` 接口更新（详见第 39 节）。录入实际到港/离港时间后，跟踪接口会自动返回最新值。

---

## 13. 航次推荐

> `GET /voyages/recommend?start_port_id=1&end_port_id=3&required_ton=500` — 需 Bearer Token

**功能说明：** 输入起运港+目的港+需求吨数，遍历所有航线找到可用航次，按各航段瓶颈剩余运力降序排列返回推荐列表。剩余运力 = 船舶最大载重 - 该航段已占吨位。结果缓存 1 分钟

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_port_id | Long | 1 | 出发港口 ID |
| end_port_id | Long | 1 | 目的港口 ID |
| required_ton | Double | 1 | 需要运输的吨数 |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "LineID": 1,
      "VesselID": 1,
      "VoyageDate": "2026-07-15",
      "VesselName": "M/V Ocean Queen",
      "LineName": "Asia-Europe Express",
      "RemainingCapacity": 49500
    }
  ]
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| LineID | Integer | 航线 ID |
| VesselID | Integer | 船舶 ID |
| VoyageDate | String | 航次日期 (yyyy-MM-dd) |
| VesselName | String | 船舶名称 |
| LineName | String | 航线名称 |
| RemainingCapacity | Double | 该段剩余可用运力（吨），必须 >= required_ton |

注意：字段名采用 PascalCase（首字母大写），与其他接口的 snake_case 不同。RemainingCapacity 表示所选段上剩余的最小运力。结果缓存 1 分钟。

---

## 14. 查询港口列表

> `GET /ports?page=1&page_size=20` — 需 Bearer Token

**功能说明：** 分页查询所有港口，可选按城市 ID 筛选。结果缓存 10 分钟

### 请求参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | Integer | 0 | 1 | 页码 |
| page_size | Integer | 0 | 20 | 每页条数（最大 100） |
| city_id | Long | 0 | | 按城市筛选（可选参数） |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "port_id": 1,
      "port_name": "Shanghai Port",
      "port_code": "CNSHA",
      "city_id": 1,
      "latitude": 31.2304,
      "longitude": 121.4737,
      "port_type": "Sea Port",
      "max_draft_meter": 15.5,
      "create_time": "2026-07-03T12:00:00Z",
      "update_time": "2026-07-03T12:00:00Z",
      "delete_time": null,
      "city": null
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 3,
    "total_pages": 1
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| port_id | Integer | 港口 ID |
| port_name | String | 港口名称 |
| port_code | String | 港口代码（如 CNSHA） |
| city_id | Integer | 所属城市 ID |
| latitude | Double | 纬度 |
| longitude | Double | 经度 |
| port_type | String | 港口类型，如 "Sea Port" |
| max_draft_meter | Double | 最大吃水深度（米） |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |
| delete_time | String | 删除时间（null 表示未删除） |
| city | Object | 城市信息（列表接口中为 null） |

**meta（分页信息）同第 9 节。**

---

## 15. 查询单个港口

> `GET /ports/{id}` — 需 Bearer Token

**功能说明：** 根据 ID 查询单个港口详情，附带所属城市信息（城市名、国家、国家代码）

### 请求参数

无请求体，{id} 为港口 ID。

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "port_id": 1,
    "port_name": "Shanghai Port",
    "port_code": "CNSHA",
    "city_id": 1,
    "latitude": 31.2304,
    "longitude": 121.4737,
    "port_type": "Sea Port",
    "max_draft_meter": 15.5,
    "create_time": "2026-07-03T12:00:00Z",
    "update_time": "2026-07-03T12:00:00Z",
    "city": {
      "city_id": 1,
      "city_name": "Shanghai",
      "country": "China",
      "country_code": "CN"
    }
  }
}
```

### 返回数据字段

**港口字段（同第 14 节，不含 delete_time）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| port_id | Integer | 港口 ID |
| port_name | String | 港口名称 |
| port_code | String | 港口代码 |
| city_id | Integer | 所属城市 ID |
| latitude | Double | 纬度 |
| longitude | Double | 经度 |
| port_type | String | 港口类型 |
| max_draft_meter | Double | 最大吃水深度（米） |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |

**city（城市信息）子字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| city_id | Integer | 城市 ID |
| city_name | String | 城市名称 |
| country | String | 国家名称 |
| country_code | String | 国家代码（如 CN） |

---

## 16. 查询船舶列表

> `GET /vessels?page=1&page_size=20` — 需 Bearer Token

**功能说明：** 分页查询所有船舶，可选按所属船公司 ID 筛选。结果缓存 10 分钟

### 请求参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | Integer | 0 | 1 | 页码 |
| page_size | Integer | 0 | 20 | 每页条数（最大 100） |
| shipping_company_id | Long | 0 | | 按船公司筛选（可选） |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "vessel_id": 1,
      "vessel_name": "M/V Ocean Queen",
      "call_sign": "OCQN",
      "imo_number": "IMO9876543",
      "vessel_type": "Container Ship",
      "max_deadweight_ton": 50000,
      "gross_tonnage": 35000,
      "net_tonnage": 25000,
      "draft_meter": 12.5,
      "speed_knot": 22,
      "container_teu": 5000,
      "is_available": 1,
      "shipping_company_id": 1,
      "create_time": "2026-07-03T12:00:00Z",
      "update_time": "2026-07-03T12:00:00Z",
      "delete_time": null,
      "shipping_company": null
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| vessel_id | Integer | 船舶 ID |
| vessel_name | String | 船舶名称 |
| call_sign | String | 船舶呼号 |
| imo_number | String | IMO 编号（国际海事组织编号） |
| vessel_type | String | 船舶类型，如 "Container Ship" |
| max_deadweight_ton | Double | 最大载重吨 (DWT) |
| gross_tonnage | Double | 总吨位 |
| net_tonnage | Double | 净吨位 |
| draft_meter | Double | 吃水深度（米） |
| speed_knot | Double | 航速（节） |
| container_teu | Integer | 集装箱容量 (TEU) |
| is_available | Integer | 是否可用：1=可用，0=不可用 |
| shipping_company_id | Integer | 所属船公司 ID |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |
| delete_time | String | 删除时间（null 表示未删除） |
| shipping_company | Object | 船公司信息（列表接口中为 null） |

**meta（分页信息）同第 9 节。**

---

## 17. 查询单个船舶

> `GET /vessels/{id}` — 需 Bearer Token

**功能说明：** 根据 ID 查询单个船舶详情，附带所属船公司信息

### 请求参数

无请求体，{id} 为船舶 ID。

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "vessel_id": 1,
    "vessel_name": "M/V Ocean Queen",
    "call_sign": "OCQN",
    "imo_number": "IMO9876543",
    "vessel_type": "Container Ship",
    "max_deadweight_ton": 50000,
    "gross_tonnage": 35000,
    "net_tonnage": 25000,
    "draft_meter": 12.5,
    "speed_knot": 22,
    "container_teu": 5000,
    "is_available": 1,
    "shipping_company_id": 1,
    "shipping_company": {
      "company_id": 1,
      "company_name": "Ocean Shipping Inc."
    }
  }
}
```

### 返回数据字段

**船舶字段（同第 16 节，不含 delete_time）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| vessel_id | Integer | 船舶 ID |
| vessel_name | String | 船舶名称 |
| call_sign | String | 船舶呼号 |
| imo_number | String | IMO 编号 |
| vessel_type | String | 船舶类型 |
| max_deadweight_ton | Double | 最大载重吨 |
| gross_tonnage | Double | 总吨位 |
| net_tonnage | Double | 净吨位 |
| draft_meter | Double | 吃水深度（米） |
| speed_knot | Double | 航速（节） |
| container_teu | Integer | 集装箱容量 (TEU) |
| is_available | Integer | 是否可用 |
| shipping_company_id | Integer | 所属船公司 ID |

**shipping_company（船公司信息）子字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| company_id | Integer | 船公司 ID |
| company_name | String | 船公司名称 |

---

## 18. 查询航线列表

> `GET /shipping-lines?page=1&page_size=20` — 需 Bearer Token

**功能说明：** 分页查询所有航线，包含航线名称、距离、起止港、港口序列等信息。结果缓存 10 分钟

### 请求参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | Integer | 0 | 1 | 页码 |
| page_size | Integer | 0 | 20 | 每页条数（最大 100） |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "line_id": 1,
      "line_name": "Asia-Europe Express",
      "shipping_company_id": 1,
      "port_sequence": "[1,2,3]",
      "total_distance_nm": 8500,
      "departure_port_name": "Shanghai Port",
      "destination_port_name": "Rotterdam Port",
      "description": "Asia to Europe direct route via Singapore",
      "create_time": "2026-07-03T12:00:00Z",
      "update_time": "2026-07-03T12:00:00Z",
      "delete_time": null,
      "shipping_company": null
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| line_id | Integer | 航线 ID |
| line_name | String | 航线名称 |
| shipping_company_id | Integer | 所属船公司 ID |
| port_sequence | String | 港口顺序（JSON 字符串，如 "[1,2,3]"，前端需 JSON.parse） |
| total_distance_nm | Double | 总距离（海里） |
| departure_port_name | String | 出发港口名称 |
| destination_port_name | String | 目的港口名称 |
| description | String | 航线描述 |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |
| delete_time | String | 删除时间（null 表示未删除） |
| shipping_company | Object | 船公司信息（列表接口中为 null） |

注意：port_sequence 是 JSON 字符串，前端需 JSON.parse("[1,2,3]") 得到数组。

**meta（分页信息）同第 9 节。**

---

## 19. 查询单个航线

> `GET /shipping-lines/{id}` — 需 Bearer Token

**功能说明：** 根据 ID 查询单个航线详情，附带所属船公司信息和港口序列（JSON 字符串格式）

### 请求参数

无请求体，{id} 为航线 ID。

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "line_id": 1,
    "line_name": "Asia-Europe Express",
    "shipping_company_id": 1,
    "port_sequence": "[1,2,3]",
    "total_distance_nm": 8500,
    "departure_port_name": "Shanghai Port",
    "destination_port_name": "Rotterdam Port",
    "description": "Asia to Europe direct route via Singapore",
    "shipping_company": {
      "company_id": 1,
      "company_name": "Ocean Shipping Inc."
    }
  }
}
```

### 返回数据字段

**航线字段（同第 18 节，不含 delete_time/audit）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| line_id | Integer | 航线 ID |
| line_name | String | 航线名称 |
| shipping_company_id | Integer | 所属船公司 ID |
| port_sequence | String | 港口顺序（JSON 字符串） |
| total_distance_nm | Double | 总距离（海里） |
| departure_port_name | String | 出发港口名称 |
| destination_port_name | String | 目的港口名称 |
| description | String | 航线描述 |

**shipping_company（船公司信息）子字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| company_id | Integer | 船公司 ID |
| company_name | String | 船公司名称 |

---

## 20. 查询航线港口序列

> `GET /shipping-lines/{id}/port-sequence` — 需 Bearer Token

**功能说明：** 根据航线 ID 查询该航线的港口顺序列表，返回整数数组格式的港口 ID 数组（按航线顺序排列）

### 请求参数

无请求体，{id} 为航线 ID。

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "port_sequence": [1, 2, 3]
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| port_sequence | Integer[] | 港口 ID 数组，按航线顺序排列 |

返回纯整数数组（港口 ID），可按 ID 查询港口详情获取名称和坐标。

---

## 21. 查询通知列表

> `GET /notifications?page=1&page_size=20` — 需 Bearer Token

**功能说明：** 分页查询当前用户的通知列表（按用户 ID + 角色过滤），包含通知类型、标题、内容、创建时间、已读/未读状态

### 请求参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | Integer | 0 | 1 | 页码 |
| page_size | Integer | 0 | 20 | 每页条数 |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "notif_1741234567890",
      "type": "order_created",
      "user_id": 1,
      "user_role": "shipper",
      "title": "订单创建通知",
      "content": "您的订单 ORD001 已创建成功",
      "data": null,
      "create_time": "2026-07-03T12:00:00Z",
      "read": false
    }
  ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| id | String | 通知唯一标识 |
| type | String | 通知类型：order_created / order_cancelled / status_changed |
| user_id | Integer | 目标用户 ID |
| user_role | String | 目标角色：shipper / shipping / admin |
| title | String | 通知标题 |
| content | String | 通知内容 |
| data | Object | 附加数据（可为 null），含 email 则发邮件，含 phone 则发短信 |
| create_time | String | 创建时间 (ISO 8601) |
| read | Boolean | 是否已读 |

注意：通知存储在内存中，服务重启后会丢失。

**meta（分页信息）同第 9 节。**

---

## 22. 标记通知已读

> `PUT /notifications/{id}/read` — 需 Bearer Token

**功能说明：** 将指定通知 ID 标记为已读状态

### 请求参数

无请求体，{id} 为通知 ID。

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "marked as read"
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| message | String | 操作结果提示信息 |

---

## 23. 管理员创建管理员

> `POST /admin/register` — 需 Bearer Token + role=admin

**功能说明：** 现有管理员创建新的管理员账号。可指定用户名、密码、真实姓名、角色级别（1=超级管理员，2=普通管理员）。仅 admin 角色可调用

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | String | 1 | 用户名 |
| password | String | 1 | 密码，至少 6 位 |
| real_name | String | 0 | 真实姓名 |
| role | Integer | 0 | 1=超级管理员, 2=普通管理员（默认 2） |

### 请求示例

```json
{
  "username": "admin2",
  "password": "admin123",
  "real_name": "管理员二号",
  "role": 2
}
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "admin_id": 2,
    "username": "admin2",
    "real_name": "管理员二号",
    "role": 2,
    "create_time": "2026-07-03T12:00:00Z",
    "update_time": "2026-07-03T12:00:00Z",
    "delete_time": null
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| admin_id | Integer | 管理员 ID |
| username | String | 用户名 |
| real_name | String | 真实姓名（可为 null） |
| role | Integer | 角色：1=超级管理员, 2=普通管理员 |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |
| delete_time | String | 删除时间（null 表示未删除） |

---

## 24. 管理员修改密码

> `POST /admin/password/{id}` — 需 Bearer Token + role=admin

**功能说明：** 修改指定管理员账号的密码。需提供旧密码验证，新密码 bcrypt 加密后更新。仅 admin 角色可调用

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| old_password | String | 1 | 旧密码 |
| new_password | String | 1 | 新密码，至少 6 位 |

### 请求示例

```json
{
  "old_password": "admin123",
  "new_password": "newadmin456"
}
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "password updated"
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| message | String | 操作结果提示信息 |

---

## 25. 管理员发送通知

> `POST /admin/notifications` — 需 Bearer Token + role=admin

**功能说明：** 管理员向指定用户发送通知。可指定目标用户 ID、角色（shipper/shipping/admin）、通知类型、标题、内容。可选附带 email/phone 字段触发真实邮件或短信发送。仅 admin 角色可调用

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | Long | 1 | 目标用户 ID |
| role | String | 1 | 目标角色：shipper / shipping / admin |
| type | String | 1 | 通知类型：order_created / order_cancelled / status_changed |
| title | String | 1 | 标题 |
| content | String | 1 | 内容 |
| data | Map | 0 | 可选，含 email 字段发邮件，含 phone 字段发短信 |

### 请求示例

```json
{
  "user_id": 1,
  "role": "shipper",
  "type": "order_created",
  "title": "通知标题",
  "content": "通知内容",
  "data": {
    "email": "user@example.com",
    "phone": "13800138000"
  }
}
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "notification sent"
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| message | String | 操作结果提示信息 |

---

## 26. 订单统计报表

> `GET /reports/orders?start_date=2026-01-01&end_date=2026-07-04` — 需 Bearer Token

**功能说明：** 按日期范围统计订单数据，返回总订单数、总重量/体积/运费金额、已完成/已取消/运输中订单数量

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_date | String | 1 | 开始日期 yyyy-MM-dd |
| end_date | String | 1 | 结束日期 yyyy-MM-dd |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_orders": 50,
    "total_weight": 25000,
    "total_volume": 10000,
    "total_cost": 10625000,
    "completed": 30,
    "cancelled": 5,
    "in_transit": 15
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| total_orders | Integer | 订单总数 |
| total_weight | Double | 总重量（吨） |
| total_volume | Double | 总体积（立方米） |
| total_cost | Double | 总运费金额 |
| completed | Integer | 已完成订单数 |
| cancelled | Integer | 已取消订单数 |
| in_transit | Integer | 运输中订单数 |

---

## 27. 航次利用率报表

> `GET /reports/voyage-utilization?line_id=1&vessel_id=1&voyage_date=2026-07-15` — 需 Bearer Token

**功能说明：** 查询指定航次（航线+船舶+日期）的运力利用率，返回船舶最大载重、已占用吨位、利用率百分比。利用率 = 已占吨位 / 最大载重 × 100

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| line_id | Long | 1 | 航线 ID |
| vessel_id | Long | 1 | 船舶 ID |
| voyage_date | String | 1 | 航次日期 yyyy-MM-dd |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "max_deadweight_ton": 50000,
    "used_ton": 500,
    "utilization_rate": 1.0
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| max_deadweight_ton | Double | 最大运力（吨） |
| used_ton | Double | 已占用运力（吨） |
| utilization_rate | Double | 利用率（百分比值，如 1.0 表示 1%） |

utilization_rate = (used_ton / max_deadweight_ton) x 100，百分比值。

---

## 28. Excel 导出

> `GET /export/{type}` — 需 Bearer Token

**功能说明：** 将指定数据表导出为 xlsx 格式的 Excel 文件。支持导出港口、船舶、航线、订单。导出文件同时自动保存到服务器 backend/excel/ 目录。订单导出需指定 shipper_company_id

### 接口列表

| 接口路径 | 文件名 | 说明 |
|---------|--------|------|
| /export/ports | ports.xlsx | 导出所有港口 |
| /export/vessels | vessels.xlsx | 导出所有船舶 |
| /export/shipping-lines | shipping_lines.xlsx | 导出所有航线 |
| /export/orders?shipper_company_id={id} | orders.xlsx | 导出指定货主的订单 |

### 响应

返回 Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet，浏览器直接触发下载。

---

## 29. Excel 导入

> `POST /import/{type}` — 需 Bearer Token

**功能说明：** 通过上传 xlsx 文件批量导入数据。支持导入港口、船舶、航线。上传文件需符合导出格式的列顺序和表头

### 接口列表

| 接口路径 | 说明 |
|---------|------|
| /import/ports | 批量导入港口 |
| /import/vessels | 批量导入船舶 |
| /import/shipping-lines | 批量导入航线 |

### 请求参数

multipart/form-data 格式，字段名 file，上传 .xlsx 文件。

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "imported": 10
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| imported | Integer | 成功导入的记录数 |

Excel 第一行必须是表头，列顺序与导出格式一致。

---

## 30. WebSocket

> `ws://localhost:8080/ws?token={access_token}`

**功能说明：** WebSocket 实时推送连接。建立连接后，当订单状态变更时（通过 PUT /orders/{id}/status），服务器自动向该订单货主推送状态更新消息。断线后建议 3 秒重连

### 连接说明

| 项目 | 说明 |
|------|------|
| 路径 | /ws（不在 /api/v1 下） |
| 认证 | URL 参数携带 access_token |
| 协议 | WebSocket |

### 推送消息格式

```json
{
  "type": "order_status_update",
  "order_id": 1,
  "status": 2,
  "timestamp": "2026-07-03T12:00:00Z"
}
```

### 推送字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| type | String | 消息类型：order_status_update(订单状态更新) |
| order_id | Integer | 订单 ID |
| status | Integer | 更新后的订单状态 |
| timestamp | String | 推送时间 (ISO 8601) |

触发时机：当调用 PUT /orders/{id}/status 更新订单状态时，自动推送。

### 前端建议

```javascript
const ws = new WebSocket(`ws://localhost:8080/ws?token=${token}`)

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data)
  if (msg.type === 'order_status_update') {
    // 更新对应订单的状态显示
  }
}

ws.onclose = () => {
  // 断线 3 秒后重连
  setTimeout(() => reconnect(), 3000)
}
```

---

## 31. 健康检查

> `GET http://localhost:8080/health`

**功能说明：** 服务存活检测。返回 {"status":"ok"}。注意：此接口不遵循统一 {code,message,data} 格式，仅用于负载均衡健康检查

### 响应示例

```json
{
  "status": "ok"
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| status | String | 服务状态，正常时为 "ok" |

---

## 32. 管理员删除货主公司

> `POST /admin/shipper/{id}/delete` — 需 Bearer Token + role=admin

**功能说明：** 管理员软删除货主公司账号。设置 delete_time 后该账号无法登录（查询被 NotDeleted scope 过滤），同时释放用户名唯一约束。仅 admin 角色可调用

### 请求参数

无请求体，{id} 为货主公司 ID。

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "shipper company deleted"
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| message | String | 操作结果提示信息 |

后端操作：软删除货主公司记录（设置 delete_time）。

---

## 33. 管理员删除船运公司

> `POST /admin/shipping/{id}/delete` — 需 Bearer Token + role=admin

**功能说明：** 管理员软删除船运公司账号。设置 delete_time 后该账号无法登录（查询被 NotDeleted scope 过滤），同时释放用户名唯一约束。仅 admin 角色可调用

### 请求参数

无请求体，{id} 为船运公司 ID。

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "shipping company deleted"
  }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| message | String | 操作结果提示信息 |

后端操作：软删除船运公司记录（设置 delete_time）。

---

---

## 42. 虚拟支付

> `POST /orders/{id}/pay` — 需 Bearer Token

**功能说明：** 货主对指定订单进行虚拟支付，将支付状态更新为已支付

### 请求参数

无请求体，{id} 为订单 ID。

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "payment successful"
  }
}
```

---

## 43. 航线申请（创建航次靠泊记录）

> `POST /voyages/berthing` — 需 Bearer Token

**功能说明：** 船公司创建航次靠泊记录。需指定航线、船舶、日期、港口、计划到港/离港时间

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| line_id | Long | 1 | 航线 ID |
| vessel_id | Long | 1 | 船舶 ID |
| voyage_date | String | 1 | 航次日期 yyyy-MM-dd |
| sequence_no | Integer | 0 | 港口序号 |
| port_id | Long | 1 | 港口 ID |
| berth_id | Long | 0 | 泊位 ID |
| planned_arrival_time | String | 0 | 计划到达时间 |
| planned_departure_time | String | 0 | 计划离港时间 |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "berthing_id": 4,
    "line_id": 1,
    "vessel_id": 1,
    "planned_arrival_time": "2026-07-15T08:00:00Z"
  }
}
```

---

## 44. 管理员查询货物列表

> `GET /admin/cargo/list?page=1&page_size=20` — 需 Bearer Token + role=admin

**功能说明：** 管理员分页查询所有货物的运输记录，含货物名称、类型、重量、关联订单等信息。仅 admin 角色可调用

### 请求参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | Integer | 0 | 1 | 页码 |
| page_size | Integer | 0 | 20 | 每页条数（最大 100） |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "detail_id": 1,
      "order_id": 1,
      "cargo_name": "钢材",
      "cargo_type": "bulk",
      "quantity": 50,
      "weight_ton": 250,
      "volume_cubic_meter": 100,
      "unit_price": 60,
      "subtotal": 3000,
      "create_time": "2026-07-07T12:00:00Z",
      "order": {
        "order_id": 1,
        "order_no": "ORD20260707xxxx"
      }
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| detail_id | Integer | 货物明细 ID |
| order_id | Integer | 关联订单 ID |
| cargo_name | String | 货物名称 |
| cargo_type | String | 货物类型：bulk/container/liquid |
| quantity | Double | 数量 |
| weight_ton | Double | 重量（吨） |
| volume_cubic_meter | Double | 体积（立方米） |
| unit_price | Double | 单价 |
| subtotal | Double | 小计金额 |
| order | Object | 关联的订单信息（含 order_id、order_no） |

---

## 附录：完整 API 路径速查

### 公开（无需 Token）

| 方法 | 路径 | 功能说明 |
|------|------|----------|
| GET | /health | 服务存活检测，返回 {"status":"ok"}，用于负载均衡健康检查 |
| GET | /ws | WebSocket 连接（需 URL 参数 ?token=access_token），用于接收订单状态变更实时推送 |
| POST | /api/v1/auth/login | 用户登录：三种角色（shipper/shipping/admin）用用户名+密码登录，返回 access_token(15min) + refresh_token(7d) |
| POST | /api/v1/auth/refresh | 令牌续期：用 refresh_token 换取新的 access_token，无需重新登录 |
| POST | /api/v1/shipper/register | 货主公司注册：创建货主账号，公司名+用户名+密码，密码 bcrypt 加密存储 |
| POST | /api/v1/shipping/register | 船运公司注册：创建船公司账号，公司名+用户名+密码，密码 bcrypt 加密存储 |

### 受保护（需 Bearer Token）

| 方法 | 路径 | 功能说明 |
|------|------|----------|
| POST | /api/v1/shipper/password/{id} | 货主修改密码：需验证旧密码，新密码 bcrypt 加密后更新。{id} 必须等于当前登录用户 ID |
| POST | /api/v1/shipping/password/{id} | 船公司修改密码：同上，需验证旧密码 |
| POST | /api/v1/orders | 创建订单：选择航次/船舶/起止港，支持多货物。自动计算运费、校验运力（GET_LOCK 防超卖）、写入多表事务、缓存清除 |
| GET | /api/v1/orders/{id} | 订单详情：返回订单主信息 + 货物明细 + 关联的装/卸货单 + 起止港信息 |
| POST | /api/v1/orders/{id}/cancel | 取消订单：软删除订单及货物，释放航段运力占用，清除推荐缓存 |
| POST | /api/v1/orders/{id}/pay | 虚拟支付：将订单支付状态更新为已支付 |
| POST | /api/v1/voyages/berthing | 航线申请：船公司创建航次靠泊记录 |
| PUT | /api/v1/orders/{id}/status | 更新订单状态：按状态机规则转换（0草稿→1已确认→2运输中→3已完成，任何状态→4已取消），变更后 WebSocket 推送 |
| GET | /api/v1/orders | 订单列表：按货主 ID 分页查询，支持按创建时间/订单号/重量/状态排序。shipper 角色只能查自己的订单 |
| GET | /api/v1/orders/{id}/tracking | 货物跟踪：返回装/卸货时间、起止港靠泊计划/实际时间、船舶名称、航线名称等全程跟踪信息 |
| GET | /api/v1/voyages/recommend | 航次推荐：输入起止港+需求吨数，返回按剩余运力降序排列的可用航次，结果缓存 1 分钟 |
| GET | /api/v1/cities | 城市列表：分页查询所有城市，供货主选择出发/目的城市 |
| GET | /api/v1/ports | 港口列表：分页查询所有港口，可选按城市筛选，结果缓存 10 分钟 |
| GET | /api/v1/ports/{id} | 港口详情：返回港口基本信息 + 所属城市信息 |
| GET | /api/v1/vessels | 船舶列表：分页查询所有船舶，可选按船公司筛选，结果缓存 10 分钟 |
| GET | /api/v1/vessels/{id} | 船舶详情：返回船舶基本信息 + 所属船公司信息 |
| GET | /api/v1/shipping-lines | 航线列表：分页查询所有航线，结果缓存 10 分钟 |
| GET | /api/v1/shipping-lines/{id} | 航线详情：返回航线基本信息 + port_sequence + 所属船公司 |
| GET | /api/v1/shipping-lines/{id}/port-sequence | 航线港口序列：返回整数数组格式的港口 ID 顺序列表 |
| GET | /api/v1/export/ports | 导出港口 Excel：下载 xlsx 文件，自动保存到服务器 backend/excel/ 目录 |
| POST | /api/v1/import/ports | 导入港口 Excel：上传 xlsx 文件，批量导入港口数据 |
| GET | /api/v1/export/vessels | 导出船舶 Excel：下载 xlsx 文件，自动保存到服务器 |
| POST | /api/v1/import/vessels | 导入船舶 Excel：上传 xlsx 文件，批量导入船舶数据 |
| GET | /api/v1/export/shipping-lines | 导出航线 Excel：下载 xlsx 文件，自动保存到服务器 |
| POST | /api/v1/import/shipping-lines | 导入航线 Excel：上传 xlsx 文件，批量导入航线数据 |
| GET | /api/v1/export/orders | 导出订单 Excel：需指定 shipper_company_id，下载 xlsx 文件 |
| GET | /api/v1/notifications | 通知列表：分页查询当前用户的通知（按角色+用户 ID 过滤），含已读/未读状态 |
| PUT | /api/v1/notifications/{id}/read | 标记已读：将指定通知标记为已读 |
| GET | /api/v1/reports/orders | 订单统计报表：按日期范围统计总订单数、总重量/体积/运费、各状态数量 |
| GET | /api/v1/reports/voyage-utilization | 航次利用率报表：查询某航次的船舶最大载重、已占吨位、利用率百分比 |
| PUT | /api/v1/berthings/{id}/actual-times | 更新靠泊实际时间：录入船舶实际到港/离港时间，更新后订单跟踪接口同步生效 |
| GET | /api/v1/shipping-companies | 船公司列表：分页查询所有船公司，供货主查看可选的运输服务商 |

### 管理员（需 Token + role=admin）

| 方法 | 路径 | 功能说明 |
|------|------|----------|
| POST | /api/v1/admin/register | 创建管理员：新建管理员账号（需已有 admin 权限），可指定角色级别 |
| POST | /api/v1/admin/password/{id} | 管理员改密：修改指定管理员的密码（需提供旧密码） |
| POST | /api/v1/admin/notifications | 发送通知：向指定用户（三种角色均可）发送通知，可选附带 email/SMS 发送 |
| GET | /api/v1/admin/list | 管理员列表：分页查询所有管理员账号 |
| GET | /api/v1/admin/shipper/list | 货主公司列表：分页查询所有货主公司 |
| GET | /api/v1/admin/shipping/list | 船公司列表：分页查询所有船公司 |
| POST | /api/v1/admin/shipper/{id}/update | 更新货主公司：修改货主公司名称、联系人、账号状态等 |
| POST | /api/v1/admin/shipping/{id}/update | 更新船公司：修改船公司名称、联系人、账号状态等 |
| POST | /api/v1/admin/shipper/{id}/delete | 删除货主公司：软删除（设置 delete_time），被删账号无法登录，释放用户名唯一约束 |
| POST | /api/v1/admin/shipping/{id}/delete | 删除船运公司：软删除（设置 delete_time），被删账号无法登录，释放用户名唯一约束 |
| POST | /api/v1/admin/ports | 创建港口：新增港口信息 |
| PUT | /api/v1/admin/ports/{id} | 更新港口：修改港口名称、城市等 |
| DELETE | /api/v1/admin/ports/{id} | 删除港口：软删除 |
| POST | /api/v1/admin/vessels | 创建船舶：新增船舶信息 |
| PUT | /api/v1/admin/vessels/{id} | 更新船舶：修改船舶参数 |
| DELETE | /api/v1/admin/vessels/{id} | 删除船舶：软删除 |
| POST | /api/v1/admin/shipping-lines | 创建航线：新增航线信息 |
| PUT | /api/v1/admin/shipping-lines/{id} | 更新航线：修改航线信息 |
| DELETE | /api/v1/admin/shipping-lines/{id} | 删除航线：软删除 |
| GET | /api/v1/admin/cargo/list | 货物列表：分页查询所有货物的运输记录 |

---

## 34. 管理员查询货主公司列表

> `GET /admin/shipper/list?page=1&page_size=20` — 需 Bearer Token + role=admin

**功能说明：** 管理员分页查询所有货主公司列表，包含公司基本信息、账号状态。仅 admin 角色可调用

### 请求参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | Integer | 0 | 1 | 页码 |
| page_size | Integer | 0 | 20 | 每页条数（最大 100） |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "company_id": 1,
      "company_name": "Global Trade Co.",
      "unified_social_credit_code": null,
      "legal_representative": null,
      "contact_phone": null,
      "address": null,
      "login_username": "test001",
      "account_status": 1,
      "create_time": "2026-07-03T12:00:00Z",
      "update_time": "2026-07-03T12:00:00Z",
      "delete_time": null
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| company_id | Integer | 货主公司 ID |
| company_name | String | 公司名称 |
| unified_social_credit_code | String | 统一社会信用代码（可为 null） |
| legal_representative | String | 法定代表人（可为 null） |
| contact_phone | String | 联系电话（可为 null） |
| address | String | 公司地址（可为 null） |
| login_username | String | 登录用户名 |
| account_status | Integer | 账户状态：1=正常，0=禁用 |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |
| delete_time | String | 删除时间（null 表示未删除） |

**meta（分页信息）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| page | Integer | 当前页码 |
| page_size | Integer | 每页条数 |
| total | Integer | 总记录数 |
| total_pages | Integer | 总页数 |

---

## 35. 管理员查询船公司列表

> `GET /admin/shipping/list?page=1&page_size=20` — 需 Bearer Token + role=admin

**功能说明：** 管理员分页查询所有船公司列表。仅 admin 角色可调用

### 请求参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | Integer | 0 | 1 | 页码 |
| page_size | Integer | 0 | 20 | 每页条数（最大 100） |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "company_id": 1,
      "company_name": "Oceanic Shipping Co.",
      "unified_social_credit_code": "SHIP20240001",
      "contact_person": "John Smith",
      "contact_phone": "+65-12345678",
      "address": "12 Harbor Road, Singapore",
      "login_username": "shipping001",
      "account_status": 1,
      "create_time": "2026-07-03T12:00:00Z",
      "update_time": "2026-07-03T12:00:00Z",
      "delete_time": null
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| company_id | Integer | 船公司 ID |
| company_name | String | 公司名称 |
| unified_social_credit_code | String | 统一社会信用代码（可为 null） |
| contact_person | String | 联系人（可为 null） |
| contact_phone | String | 联系电话（可为 null） |
| address | String | 公司地址（可为 null） |
| login_username | String | 登录用户名 |
| account_status | Integer | 账户状态：1=正常，0=禁用 |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |
| delete_time | String | 删除时间（null 表示未删除） |

**meta（分页信息）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| page | Integer | 当前页码 |
| page_size | Integer | 每页条数 |
| total | Integer | 总记录数 |
| total_pages | Integer | 总页数 |

---

## 36. 管理员查询管理员列表

> `GET /admin/list?page=1&page_size=20` — 需 Bearer Token + role=admin

**功能说明：** 管理员分页查询所有管理员账号列表。仅 admin 角色可调用

请求参数同第 34 节。

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "admin_id": 1,
      "username": "admin",
      "real_name": "System Admin",
      "role": 1,
      "create_time": "2026-07-03T12:00:00Z",
      "update_time": "2026-07-03T12:00:00Z",
      "delete_time": null
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| admin_id | Integer | 管理员 ID |
| username | String | 用户名 |
| real_name | String | 真实姓名（可为 null） |
| role | Integer | 1=超级管理员, 2=普通管理员 |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |
| delete_time | String | 删除时间（null 表示未删除） |

---

## 37. 管理员更新货主公司

> `POST /admin/shipper/{id}/update` — 需 Bearer Token + role=admin

**功能说明：** 管理员修改货主公司信息（公司名称、联系人、账号状态等）。仅 admin 角色可调用

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| company_name | String | 0 | 公司名称 |
| unified_social_credit_code | String | 0 | 统一社会信用代码 |
| legal_representative | String | 0 | 法定代表人 |
| contact_phone | String | 0 | 联系电话 |
| address | String | 0 | 地址 |
| login_username | String | 0 | 登录用户名 |
| account_status | Integer | 0 | 账号状态：1=正常，0=禁用 |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "shipper company updated"
  }
}
```

---

## 38. 管理员更新船公司

> `POST /admin/shipping/{id}/update` — 需 Bearer Token + role=admin

**功能说明：** 管理员修改船公司信息（公司名称、联系人、账号状态等）。仅 admin 角色可调用。仅更新请求中传了的字段，未传字段保持不变

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| company_name | String | 0 | 公司名称 |
| unified_social_credit_code | String | 0 | 统一社会信用代码 |
| contact_person | String | 0 | 联系人 |
| contact_phone | String | 0 | 联系电话 |
| address | String | 0 | 地址 |
| login_username | String | 0 | 登录用户名 |
| account_status | Integer | 0 | 账号状态：1=正常，0=禁用 |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "shipping company updated"
  }
}
```

---

## 39. 更新靠泊实际时间

> `PUT /berthings/{id}/actual-times` — 需 Bearer Token

**功能说明：** 更新指定靠泊记录的实际到达时间和实际出发时间。船公司可在船舶实际到港/离港后录入真实时间。更新后订单跟踪接口的 departure_actual / arrival_actual 字段会同步更新

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| actual_arrival_time | String | 0 | 实际到达时间 (ISO 8601，可为 null） |
| actual_departure_time | String | 0 | 实际出发时间 (ISO 8601，可为 null） |

### 请求示例

```json
{
  "actual_arrival_time": "2026-07-15T07:45:00Z",
  "actual_departure_time": "2026-07-15T18:30:00Z"
}
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message": "actual times updated"
  }
}
```

---

## 40. 货主查看船公司列表

> `GET /shipping-companies?page=1&page_size=20` — 需 Bearer Token

**功能说明：** 分页查询所有船公司列表，供货主查看可选的运输服务商

### 请求参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | Integer | 0 | 1 | 页码 |
| page_size | Integer | 0 | 20 | 每页条数（最大 100） |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "company_id": 1,
      "company_name": "Oceanic Shipping Co.",
      "unified_social_credit_code": "SHIP20240001",
      "contact_person": "John Smith",
      "contact_phone": "+65-12345678",
      "address": "12 Harbor Road, Singapore",
      "login_username": "shipping001",
      "account_status": 1,
      "create_time": "2026-07-03T12:00:00Z",
      "update_time": "2026-07-03T12:00:00Z",
      "delete_time": null
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 }
}

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| company_id | Integer | 船公司 ID |
| company_name | String | 公司名称 |
| unified_social_credit_code | String | 统一社会信用代码（可为 null） |
| contact_person | String | 联系人（可为 null） |
| contact_phone | String | 联系电话（可为 null） |
| address | String | 公司地址（可为 null） |
| login_username | String | 登录用户名 |
| account_status | Integer | 账户状态：1=正常，0=禁用 |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |
| delete_time | String | 删除时间（null 表示未删除） |

**meta（分页信息）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| page | Integer | 当前页码 |
| page_size | Integer | 每页条数 |
| total | Integer | 总记录数 |
| total_pages | Integer | 总页数 |

---

## 41. 查询城市列表

> `GET /cities?page=1&page_size=20` — 需 Bearer Token

**功能说明：** 分页查询所有城市列表，包含城市名称、所属国家、国家代码等信息。货主选出发/目的城市时使用

### 请求参数

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | Integer | 0 | 1 | 页码 |
| page_size | Integer | 0 | 20 | 每页条数（最大 100） |

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "city_id": 1,
      "city_name": "Shanghai",
      "country": "China",
      "country_code": "CN",
      "timezone": "Asia/Shanghai",
      "latitude": 31.2304,
      "longitude": 121.4737,
      "create_time": "2026-07-03T12:00:00Z",
      "update_time": "2026-07-03T12:00:00Z",
      "delete_time": null
    }
  ],
  "meta": { "page": 1, "page_size": 20, "total": 3, "total_pages": 1 }
}
```

### 返回数据字段

| 字段 | 类型 | 说明 |
|------|------|------|
| city_id | Integer | 城市 ID |
| city_name | String | 城市名称 |
| country | String | 所属国家 |
| country_code | String | 国家代码（如 CN、SG） |
| timezone | String | 时区 |
| latitude | Double | 纬度 |
| longitude | Double | 经度 |
| create_time | String | 创建时间 (ISO 8601) |
| update_time | String | 更新时间 (ISO 8601) |
| delete_time | String | 删除时间（null 表示未删除） |

## 45. 管理员创建港口

> POST /admin/ports — 需 Bearer Token + role=admin

**功能说明：** 管理员新增港口信息

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| port_name | String | 1 | 港口名称 |
| port_code | String | 0 | 港口代码 |
| city_id | Long | 1 | 所属城市 ID |

### 响应示例

{ ""code"": 0, ""message"": ""success"", ""data"": { ""port_id"": 4, ""port_name"": ""Guangzhou Port"" } }

---

## 46. 管理员更新港口

> PUT /admin/ports/{id} — 需 Bearer Token + role=admin

**功能说明：** 管理员修改港口信息

### 响应示例

{ ""code"": 0, ""message"": ""success"", ""data"": { ""port_id"": 4, ""port_name"": ""Updated"" } }

---

## 47. 管理员删除港口

> DELETE /admin/ports/{id} — 需 Bearer Token + role=admin

### 响应示例

{ ""code"": 0, ""message"": ""success"", ""data"": { ""message"": ""port deleted"" } }

---

## 48. 管理员创建船舶

### 响应示例

{ ""code"": 0, ""message"": ""success"", ""data"": { ""vessel_id"": 2 } }

---

## 49. 管理员更新船舶

### 响应示例

{ ""code"": 0, ""message"": ""success"", ""data"": { ""vessel_id"": 2, ""vessel_name"": ""Updated"" } }

---

## 50. 管理员删除船舶

### 响应示例

{ ""code"": 0, ""message"": ""success"", ""data"": { ""message"": ""vessel deleted"" } }

---

## 51. 管理员创建航线

### 响应示例

{ ""code"": 0, ""message"": ""success"", ""data"": { ""line_id"": 2 } }

---

## 52. 管理员更新航线

### 响应示例

{ ""code"": 0, ""message"": ""success"", ""data"": { ""line_id"": 2 } }

---

## 53. 管理员删除航线

### 响应示例

{ ""code"": 0, ""message"": ""success"", ""data"": { ""message"": ""line deleted"" } }

---

## 附录：完整 API 路径速查
