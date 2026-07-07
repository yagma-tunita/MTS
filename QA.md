# MTS 后端系统设计问答集

---

## 目录

- [一、数据库脚本设计](#一数据库脚本设计)
- [二、后端架构设计](#二后端架构设计)
- [三、网络层设计](#三网络层设计)
- [四、基础设施设计](#四基础设施设计)
- [五、业务层设计](#五业务层设计)

---

## 一、数据库脚本设计

### Q1：脚本的整体结构是怎么组织的？

建表脚本 `tables_mysql.sql` 按**依赖倒序**编写——被依赖的表先建，保证 `REFERENCES` 时父表已存在。

```
第1层（无依赖）：city, shipper_company, shipping_company, admin
第2层（依赖第1层）：port → city, berth → port, vessel → shipping_company, shipping_line → shipping_company
第3层（依赖第2层）：voyage_cargo_note → shipping_line + vessel
                    voyage_berthing → shipping_line + vessel + port + berth
第4层（依赖第3层）：shipping_order（依赖6张表）
                    order_cargo → shipping_order
                    segment_capacity_usage → shipping_order + shipping_line + vessel + port
```

建表后三段索引，不混在 CREATE TABLE 里：
- 外键索引（每个 FK 单独建）
- 业务组合索引（2 个常用查询条件）
- 软删除索引（每个有 delete_time 的表建）

索引后置的好处：表结构清晰可读，索引变更不影响表定义。

---

### Q2：什么是软删除？为什么有些表有、有些没有？

软删除就是不真的 `DELETE FROM` 物理删除行，而是给数据行打一个"已删除"标记：

```
delete_time 为 NULL   → 数据正常、可见
delete_time 不为 NULL → 数据已"删除"、业务上不可见
```

所有查询必须带 `WHERE delete_time IS NULL`；真正的"删除"操作是 `UPDATE ... SET delete_time = NOW()`。

**为什么用软删除？**
- **可恢复**：误删了把 `delete_time` 改回 NULL 就恢复了，不需要从备份回滚
- **审计不丢**：对账、历史报表仍然能查到已删记录
- **唯一约束不冲突**：唯一索引带上 `delete_time`，被删的记录保留原值也不影响新插入

**为什么 `voyage_cargo_note` 和 `voyage_berthing` 没有 `delete_time`？**

这两张表记录的是**已经发生的物理事实**——某艘船在某天某时确实停靠了某个泊位、确实装卸了某批货。这是不可篡改的运营记录。

- 如果允许删掉一条靠泊记录，审计时发现"船在海上消失了"，没法解释
- 如果允许删掉一条装卸记录，货主说"我的货没到"，而你拿不出证据

航次数据只追加、不删除、不修改。业务数据（订单、公司信息）可以软删除；物理事实数据（航次靠泊、装卸记录）不可删除。

---

### Q3：为什么没有独立的 voyage 表？

大多数航运系统会建 `voyage` 表存 `(line_id, vessel_id, voyage_date)`，但这里没有。航次信息分散在 `voyage_berthing` 和 `voyage_cargo_note` 中，靠 `(line_id, vessel_id, voyage_date)` 三元组隐含标识。

**为什么这么做？**
- **减少一次 JOIN**：查靠泊信息时，如果中间有个 `voyage` 表，每次都要多 JOIN 一层
- **航次没有独立属性**：所有信息要么属于靠泊（什么时候到港），要么属于装卸（装了什么货）。建一张独立的 voyage 表里面只有三个字段，没什么实际意义
- **一致性负担转移给代码**：不建表意味着没有外键约束保证"存在这个航次"，业务代码要自己保证数据一致性

**代价**：没有航次级的状态字段，如果将来需要"整个航次取消"的功能，没法在一张 voyage 表上设状态，只能遍历所有靠泊和货单记录来推断。

---

### Q4：为什么用 JSON 存港口序列而不是关联表？

`shipping_line.port_sequence` 用 MySQL `JSON` 类型存 `[1001, 1005, 1008]`，表示航线途径港口的 ID 序列。

**为什么不用关联表 `line_port`？**
- **每条航线的港口数很少**（通常 3-8 个）。为这么少的数据建一张表，多一次 JOIN，不值得
- **港口序列查询频率极高**：每次下订单都要把 port_sequence 取出来解析。JSON 一次读取即可，不需要 `ORDER BY sequence_no`
- **MySQL 的 JSON 类型是二进制存储**，解析比 TEXT 快，还支持 `JSON_CONTAINS`、`JSON_SEARCH` 等原生函数

`departure_port_name` / `destination_port_name` 是冗余字段——首尾港口名直接从 JSON 推导，但为了列表查询快，单独存了一份。

---

### Q5：为什么运力要按航段拆分，而不是按订单汇总？

简单做法是把订单总吨位记在 `shipping_order.total_weight_ton`，然后按订单汇总。但这样不够精确。

**场景**：航线 上海→宁波→深圳→广州，船最大载重 1000 吨。
- 订单 A：上海→深圳，500 吨
- 订单 B：宁波→广州，600 吨

按订单汇总：`500 + 600 = 1100 > 1000`，拒绝 B。

但实际物理情况：A 占用的航段是"上海→宁波"和"宁波→深圳"。B 占用的航段是"宁波→深圳"和"深圳→广州"。重叠的只有"宁波→深圳"这一段。A 在这一段占 500 吨，所以 B 在"宁波→深圳"段还能装 500 吨，B 需要 600 吨，但重叠段只够 500 吨——这才是拒绝 B 的真正原因，而非简单地 500+600>1000。

**`segment_capacity_usage` 的方式**：订单从 A 到 C，系统拆成 A→B + B→C 两个航段，分别写入两行记录。

好处：
- **精确计算每段剩余容量**：只有航段级别的记录，才能准确判断两个订单是否在同一段上竞争运力
- **支持中途释放**：如果货在宁波卸了一半，释放了"宁波→深圳"段的吨位，后续订单就可以利用这段新增的容量
- **支持分段装载**：订单可以在不同港口装不同批次的货，每段占不同吨位

---

### Q6：为什么 shipping_order 有 6 个外键？

订单表是业务中心，关联了 6 张表：
```
shipper_company_id  → 谁下的单
city_id             → 哪个城市的货主
load_note_id        → 在哪个航次的哪个港口装货
unload_note_id      → 在哪个航次的哪个港口卸货
departure_port_id   → 起运港（冗余）
destination_port_id → 目的港（冗余）
```

`departure_port_id` 和 `destination_port_id` 是**冗余字段**——理论上可以从 `load_note_id` 和 `unload_note_id` 推导出来。但下单列表页要显示"上海→深圳"，每次 JOIN 三张表太慢。直接把港口 ID 存在订单表里，单表就能查。这两个值在订单生成后就固定了，冗余是安全的。

---

### Q7：为什么索引分三批建？

| 批次 | 类型 | 数量 | 目的 |
|---|---|---|---|
| 第一批 | 外键索引 | 16 个 | 防止 InnoDB 在修改父表时全表扫描子表 |
| 第二批 | 业务组合索引 | 2 个 | 覆盖高频查询场景 |
| 第三批 | 软删除索引 | 9 个 | 加速 `WHERE delete_time IS NULL` 过滤 |

**为什么外键要建索引？** InnoDB 在 UPDATE/DELETE 父表时，需要检查子表是否有对应的行。如果没有索引，InnoDB 只能全表扫描子表来找匹配行，这时候行锁会升级成表锁，并发性能暴跌。

**为什么软删除要建索引？** `WHERE delete_time IS NULL` 这个条件对 MySQL 来说很难优化——NULL 值在 B+ 树中通常排在一边，但如果没有索引，就需要扫全表。

---

### Q8：优化脚本为什么和建表脚本分离？

`optimizations_mysql.sql` 包含视图、存储过程、触发器，与 `tables_mysql.sql` 分开。

`tables_mysql.sql` 是 DDL（数据定义语言），定义表结构。执行一次，基本不变。
`optimizations_mysql.sql` 是辅助数据库对象，可以反复执行（`CREATE OR REPLACE`），跟业务逻辑迭代频率一致。

如果放在同一个文件，每次修改视图都要重新执行整个建表脚本，风险大。

---

### Q9：视图是做什么的？

- **`vw_order_detail`**：跨 6 张表 JOIN 拼出订单完整信息。Go 代码里已经用 GORM 做了 JOIN 查询，视图是为 DBA 或数据分析师提供的快捷查询入口
- **`vw_voyage_capacity`**：每个航次靠泊点的已装货量和剩余容量，通过子查询对 `segment_capacity_usage` 按起航港分组求和
- **`vw_port_sequence_detail`**：航线 JSON 概览，用 `JSON_LENGTH()` 算港口数量

### Q10：触发器是做什么的？

**`trg_shipping_order_before_update_delete_time`**：当订单被软删除时，自动把 `order_cargo` 里关联的货物明细也软删除。

如果没有这个触发器，订单软删除后 `order_cargo` 里的货物明细还留着 `delete_time = NULL`，变成"没有订单的货物"——**孤儿数据**。触发器保证数据一致性由数据库保证，不依赖应用程序。

触发器和 Go 代码的职责划分：
- **数据一致性归数据库（触发器）**：级联软删除，不管通过什么途径删除订单，货物都会被同步删除
- **业务逻辑归代码（Go 函数）**：状态变更写审计日志等

---

### Q11：存储过程有什么用？

- **`sp_get_voyage_remaining_capacity`**：传入航线、船、日期、起止港，输出该航段剩余容量。运力校验的逻辑在 Go 代码里用 `CapacityChecker` 做了，存储过程是备选方案，方便从数据库端直接调用
- **`sp_recommend_voyages`**：传入起运港、目的港、需求吨位，推荐可用航次。内部用 `JSON_CONTAINS` 检查港口是否在线路上，用 `JSON_SEARCH` 比较港口先后顺序

---

### Q12：整个数据库设计有哪些核心思想？

1. **软删除按数据类型精准区分**：业务数据可软删除（订单、公司信息），物理事实不可删除（航次靠泊、装卸记录）
2. **航次不建表**：用 `(航线, 船, 日期)` 三元组隐含标识，省一次 JOIN
3. **JSON 取代关联表**：港口序列用 JSON 一列搞定，简单问题简单解决
4. **航段级运力**：运力不是按订单汇总，而是按起止港对拆分，支持中途释放
5. **三段索引**：外键索引 + 业务索引 + 软删除索引，各管各的

---

## 二、后端架构设计

### Q13：为什么选 Go 而不是 Java 或 Python？

航运物流系统是**IO密集型**的——大部分时间花在等数据库、等网络、等文件读写。Go 的 goroutine 在这种场景下比 Java 的线程轻量得多（一个 goroutine 几 KB，一个 Java 线程几 MB），一台普通服务器就能支撑几千个并发连接。Java 要扛同样并发量需要更复杂的线程池调优和更大的内存。

同时 Go 编译成单个二进制文件部署——不像 Java 要装 JRE、配 classpath，也不像 Python 要装解释器和依赖包。对于可能部署在码头机房或船上等边缘服务器的场景，单文件部署的运维成本低很多。

---

### Q14：为什么是分层架构不是 DDD 也不是 MVC？

代码分五层：

```
handler → service → biz  → dao
                ↓
             websocket
```

| 层级 | 目录 | 职责 | 允许依赖 |
|---|---|---|---|
| handler | `internal/handler/` | 收 HTTP 请求、解析参数、校验格式、返回响应 | 只调 service |
| service | `internal/service/` | 编排业务流程、管理事务、组合 dao + biz | 调 dao 和 biz |
| biz | `internal/biz/` | 纯业务逻辑：规则计算、状态校验 | 不依赖任何外部 I/O |
| dao | `internal/dao/` | 数据库读写操作 | 只调 model |

越下层越稳定，越上层变化越快。`biz` 层最稳定——里面是航运业务的固定规则；`handler` 层最容易变——今天用 Gin 框架，明天可能换 Fiber，变化只影响 handler。

这种分层比 DDD 更轻（不需要聚合根、值对象等概念），比 MVC 更清楚——MVC 里 Controller 经常塞满了业务逻辑，最后变成"胖 Controller 瘦 Model"。

---

### Q15：为什么手动依赖注入不用框架？

`app/main.go` 里一行一行手动创建所有组件，没有用 wire、dig、fx 等 DI 框架。

依赖树虽然大（13 个 DAO、7 个 biz 组件、8 个 service），但层次清楚、没有循环依赖。手动写就是几十行代码，跟加一个 DI 框架的依赖和注解相比，手动写在可读性上完胜。新来的人打开 `main.go` 从上往下读一遍，就知道"系统的全部组件是怎么拼在一起的"。

---

### Q16：为什么 DAO 层要写接口？

```go
type ShippingOrderDAO interface {
    Create(order *model.ShippingOrder) error
    GetByID(id int64) (*model.ShippingOrder, error)
}
type shippingOrderDAOImpl struct { db *gorm.DB }
```

两个好处：
1. **测试时可以换实现**：单元测试里接口可以换成 mock 实现，不需要真的连数据库
2. **构造时可以换数据库**：将来某张表从 MySQL 迁移到 Redis，只要做一个新实现满足接口就行，调用方代码一行不改

---

### Q17：为什么 biz 层和 dao 层不直接通信？

`CapacityChecker.Check()` 接收的是回调函数而不是直接调用 dao：

```go
type CapacityChecker interface {
    Check(segments, maxWeight,
        occupiedGetter func(...) (float64, error),  // ← 回调
        totalWeight) (bool, float64, error)
}
```

Service 层在调用时把 DAO 的方法塞进去作为回调。这样 `CapacityChecker` 不需要知道数据从哪来，它只负责计算规则。biz 层做到了**零外部依赖**——可以把它拿到另一个用完全不同数据库的项目里，只要传给它一个回调就能直接用。

---

### Q18：为什么创建订单要用 MySQL GET_LOCK + SELECT FOR UPDATE 两层锁？

两个货主同时下单，可能同时查到"剩余容量够"，然后同时写入，结果超卖。你用了两层保护：

**第一层：MySQL 咨询锁（GET_LOCK）**
```go
ok, err := AcquireLock(tx, lockName, 10)
// lockName = "voyage_<lineID>_<vesselID>_<date>"
```
同一个航次的订单创建操作，一次只允许一个进行。锁名精确到航次，不同航次不互相阻塞。超时 10 秒。

**第二层：SELECT FOR UPDATE**
```go
SELECT 1 FROM segment_capacity_usage WHERE ... FOR UPDATE
```
锁定涉及的航段行，其他事务不能修改它们直到当前事务提交。

为什么两层？
- `GET_LOCK` 是互斥锁——一次只能一个人进
- `FOR UPDATE` 是行锁——你读的时候别人不能写

如果只用 `GET_LOCK`：拿到锁后 MySQL 连接意外断开，锁自动释放，但事务可能还在运行。
如果只用 `FOR UPDATE`：两个事务可以同时执行 SELECT FOR UPDATE，其中一个会等另一个提交后才能读到数据，但此时数据已经被改了——这就是幻读。

两层互为补充。

---

### Q19：为什么用两个 JWT token 而不是一个？

双 token 机制：access token（15 分钟过期）+ refresh token（7 天过期）。

如果只用一个 token：
- 过期时间设短了（15 分钟），用户每 15 分钟就要重新登录一次，体验极差
- 设长了（7 天），token 泄露了黑客能用一周，风险太大

双 token 的折中：access token 短命，即使泄露也只影响 15 分钟内的操作；refresh token 长命，但只用来换新的 access token，不能直接操作用户数据。

---

### Q20：为什么 WebSocket 路径放在全局中间件之前？

路由注册顺序很讲究：

```go
r.GET("/ws", wsHandler.ServeWS)    // 第一步
r.Use(middleware.Logger())          // 第二步
r.Use(middleware.Recovery())        // 第二步（后续 8 个中间件）
// ... API 路由 ...
```

Gin 的 `r.Use()` 只对**注册之后**的路由生效。WebSocket 在全局中间件之前注册，不走 Logger、Recovery、RateLimit、RequestGuard 等中间件。

为什么 WebSocket 不能走这些中间件？因为 `gorilla/websocket` 的 HTTP 升级握手是特殊请求，如果经过 `RequestGuard` 检查 body 大小、或 `RateLimit` 统计请求次数，可能会干扰升级流程。

---

### Q21：整个后端的目录结构是如何划分的？

```
app/          ← 启动入口（main.go，组装全部组件）
internal/     ← 业务逻辑（handler → service → biz → dao → model）
net/          ← 网络层（middleware, protect, router, websocket）
pkg/          ← 基础设施（config, database, jwt, cache, logger...）
cmd/          ← CLI 命令
docs/         ← Swagger 自动生成文档
sql/          ← 数据库脚本
```

不是按功能水平切分，而是按**依赖方向**垂直分层：`pkg` 谁都不依赖，`internal` 依赖 `pkg`，`net` 依赖 `internal` + `pkg`，`app` 依赖所有。

---

### Q22：main 函数的启动流程是怎样的？

```go
config → logger → database → auto_migrate → jwtSvc → DAOs → Biz
    → Services → Handlers → Router → Swagger → pprof → Server
```

线性依赖链，每个组件只依赖前面已经创建好的组件。启动时如果某个依赖创建失败，直接 panic 停掉——启动阶段失败就是失败，不可能降级运行。

---

## 三、网络层设计

### Q23：中间件的顺序为什么这么排？

```go
r.Use(middleware.Logger())         // 1. 请求日志
r.Use(middleware.Recovery())       // 2. 崩溃恢复
r.Use(middleware.NewCORS(...))     // 3. CORS
r.Use(protect.SecurityHeaders())   // 4. 安全头
r.Use(protect.Honeypot(...))       // 5. 蜜罐陷阱
r.Use(protect.IPBlocklist(...))    // 6. IP黑名单
r.Use(protect.RequestGuard(...))   // 7. 请求守卫
r.Use(middleware.RateLimit(...))   // 8. 限流
```

- **Logger 要放第一个**：只有放在最前面，才能记录到所有请求（包括后面被拒绝的）
- **Recovery 要放第二个**：放在 Logger 之后、其他中间件之前，兜住后面的 panic
- **CORS 和 SecurityHeaders 放前面**：只改响应头不改请求，不拦截请求
- **蜜罐和 IP 黑名单放中间**：代价低（查 map），早期拦截恶意请求
- **RateLimit 放最后**：前面的中间件已经过滤掉了大量恶意请求，只需要处理"正常但频率过高"的请求

---

### Q24：为什么安全层有 4 个中间件而不是 1 个？

每个中间件的职责和配置相互独立：
- `SecurityHeaders`：加 HTTP 安全头，防浏览器端的 MIME 嗅探、XSS、点击劫持
- `Honeypot`：拦截访问常见攻击路径（`/wp-admin`、`/phpmyadmin`、`/.git/config` 等）
- `IPBlocklist`：拦截指定 IP（默认关闭，留给运维扩展）
- `RequestGuard`：限制方法白名单、URL 长度（2048）、Body 大小（4MB）、User-Agent

拆开的好处：可以独立开关、独立配置默认值、独立单元测试。如果合在一起就是一个巨大的配置结构体。

---

### Q25：Honeypot 的设计理念是什么？

蜜罐故意暴露一些"看起来像漏洞"的路径，攻击者扫描到这些路径时触发告警。正常的航运系统用户永远不会访问 `/wp-admin` 或 `/phpmyadmin`。

设计理念是**不直接拒绝，而是返回 404 假装路径不存在**。这样攻击者不知道这个系统有安全防护，只会认为是自己扫到了一个没用路径。有 `LogOnly` 模式用于评估误报率。

---

### Q26：WebSocket 的 hub + client 架构是如何工作的？

```
Client A──┐
Client B──┼── Hub（中转子，全部在内存里）──→ 按 userID+role 分发
Client C──┘
```

Hub 是中央调度器，管理所有 Client 连接。每个 Client 有 ReadPump 和 WritePump 两个 goroutine——`gorilla/websocket` 不允许并发读写同一个连接，所以拆成两个，中间通过 `client.send` channel 通信。

`send` channel 的 buffer 是 256 条消息。如果用户离线太久，buffer 满了新消息会被丢弃。通知是时效性数据，旧消息用户也用不上了。

---

### Q27：为什么 WebSocket 认证用 URL query 参数？

```go
token := c.Query("token")
```

浏览器端的 WebSocket API 不允许自定义 Header（`new WebSocket("ws://host/ws?token=xxx")` 只能传 query）。如果用 `Authorization: Bearer` header，浏览器 JS 代码根本发不出来。

---

### Q28：WebSocket 的心跳机制如何工作？

```go
pongWait   = 60 * time.Second   // 60 秒没收到 pong 认为断开
pingPeriod = (pongWait * 9) / 10 // 54 秒发一次 ping
```

`pingPeriod = pongWait * 9/10` 是经验值——留 6 秒余量覆盖网络延迟，避免临界情况误判。

---

## 四、基础设施设计

### Q29：为什么响应格式统一用 code + message + data？

```json
{"code": 0, "message": "success", "data": ...}
{"code": 1001, "message": "unauthorized"}
```

HTTP status code 只有有限几个值（200、400、401、500...），但业务错误有几十种。用 `code` 字段可以精确到具体错误，前端根据 `code` 做判断而不是根据 HTTP status。

分页数据用 `meta` 而不是放在 `data` 里面——`response.data` 永远是列表内容，`response.meta` 永远是分页信息，前端做泛型处理更方便。

---

### Q30：为什么错误码用区间分类？

```
0       ：成功
1000-1999：客户端错误（请求格式不对、没权限、资源不存在）
2000+   ：服务端错误（代码 bug、数据库挂了、下游服务故障）
```

前端可以根据 code 的千位数做粗略处理：`code >= 2000` 弹"系统异常"，`code >= 1000` 弹具体错误信息。

---

### Q31：为什么 AppError 要捕捉调用栈？

`captureStack()` 用 `runtime.Callers` 记录了创建错误时的调用栈。普通业务错误不需要栈（比如"订单不存在"），但 `CodeInternal` 和 `CodeDatabaseError` 这种服务端错误，有了栈信息就能直接定位到是哪一行代码出了问题。

---

### Q32：为什么配置加载是三层覆盖？

```go
applyDefaults(cfg)         // 1. 硬编码默认值
yaml.Unmarshal(data, cfg)  // 2. config.yaml
overrideFromEnv(cfg)       // 3. 环境变量
```

- **为什么不是只用 config.yaml？** 数据库密码和 JWT 密钥是敏感信息，不能提交到 Git。环境变量可以从 Docker Secrets 或 K8s Secrets 注入
- **为什么不是只用环境变量？** 复杂结构（如 `CargoTypeFactors` 映射）在环境变量里要写成 JSON 字符串，不如 YAML 直观
- **为什么非要硬编码默认值？** 项目克隆下来 `go run ./app` 就能启动，零配置开发

环境变量解析失败会跳过而非崩溃——运维输错环境变量名，系统不会因此挂掉。

---

### Q33：为什么 GORM 配置 SkipDefaultTransaction = true？

GORM 默认在每个单表操作外包一个事务。对于单行 INSERT，这个事务是多余的——InnoDB 的单行 INSERT 本身就有原子性。关掉后每次 `db.Create` 少一次 BEGIN/COMMIT。多表操作时你用显式事务 `db.Transaction(...)`，事务不会丢。

`AllowGlobalUpdate: false` 是安全开关——禁止不带 WHERE 条件的 UPDATE/DELETE，防止手滑改全表。

---

### Q34：为什么用 slog 不用 logrus 或 zap？

Go 1.21 标准库的 `log/slog` 提供了结构化日志（JSON format + 键值对），满足全部需求：
- 结构化输出：`slog.Info("request", "method", "GET", "latency", 42)`
- 级别过滤：debug/info/warn/error
- JSON 或文本格式

不需要 zap 那种极致的性能（这个系统每秒几千日志，不是几十万），不需要 logrus 的插件生态。用标准库少一个依赖、少一个攻击面。

---

### Q35：为什么日志同时输出到 stdout 和文件？

```go
writers = append(writers, os.Stdout)
writers = append(writers, &lumberjack.Logger{...})
writer := io.MultiWriter(writers...)
```

stdout：Docker 或 K8s 环境下被容器运行时捕获，不需要额外日志收集代理。
文件（lumberjack）：按大小轮转（100MB）、按时间清理（30天）、压缩归档。

---

### Q36：为什么缓存只用于航次推荐结果？

航次推荐涉及多表 JOIN 和运力计算，是开销最大的查询之一，但查询条件相对固定。1 分钟内同条件直接返回缓存。

不给订单列表、港口列表加缓存的原因是：这些查询本身很快（单表 + 索引），而且订单列表对实时性要求高——下了立刻要在列表里看到。

`cache.DeletePrefix("voyage_rec:")` 在订单创建成功后立即清除相关缓存，保证数据新鲜度。

---

### Q37：为什么用 Sonyflake 而不用 UUID 或自增 ID？

| 方案 | 问题 |
|---|---|
| 自增 ID | 依赖数据库，分库分表时不能保证全局唯一 |
| UUID | 128 位字符串，占空间大（BIGINT 两倍），插入导致 B+ 树频繁页分裂 |

Sonyflake 生成 64 位整数，按时间递增，插入性能接近自增 ID，且不依赖数据库。

---

### Q38：为什么通知存在内存 map 里而不是数据库？

`notification_service.go` 的实现是 `map[string][]Notification`——进程重启数据就丢了。

通知的特点是时效性强、可靠性要求低。用户收到通知后通常几秒就看了，不会回查 3 天前的。如果因为重启丢了几条，用户最多就是"好像没看到那条通知"——不会导致货物损失或订单错误。

如果存数据库，每发一条通知就要 INSERT 一次，高频操作里多了不必要的数据库写入。

---

### Q39：为什么密码用 bcrypt 而不是 MD5 或 SHA256？

MD5 和 SHA256 计算速度极快——攻击者用 GPU 每秒算几十亿次。bcrypt 设计为"慢"哈希，一次计算几十毫秒，攻击者每秒只能算几千次，破解成本高了几百万倍。

---

### Q40：为什么时区固定在 UTC+8？

航运系统的所有操作（装货时间、到港时间、下单时间）都以中国时区为准。如果使用 `time.Local`，服务器部署在不同时区可能导致时间错乱。固定 UTC+8 消除了这种不确定性。

---

## 五、业务层设计

### Q41：biz 层包含哪些业务逻辑？

`internal/biz/` 包包含以下模块，**零依赖 DB、HTTP 或外部 I/O**：

| 模块 | 职责 |
|---|---|
| `PortSequenceParser` | 解析 JSON 港口序列为 `[]int64` |
| `SegmentCalculator` | 根据起止港口计算途经的所有航段 |
| `CapacityChecker` | 校验所有航段是否不超过最大载重 |
| `CostCalculator` | 计算运费（吨×距离×费率×货种系数） |
| `OrderNoGenerator` | 生成订单号 `ORD<YYYYMMDD><8hex>` |
| `OrderStateMachine` | 状态转换矩阵：草稿→确认→运输中→完成/取消 |
| `VoyageRecommender` | 推荐可用航次并排序 |

---

### Q42：Service 层的 CreateOrder 流程是怎样的？

创建订单是最复杂的事务操作，流程如下：

```
1. 校验 cargo 非空
2. CostCalculator 计算货物小计
3. 查询 vessel 获取 max_deadweight_ton
4. 查询 shipping_line 获取 port_sequence + total_distance_nm
5. CostCalculator 计算运费（吨 × 距离 × 基础费率 × 货种系数）
6. PortSequenceParser 解析 JSON 为 portIDs
7. SegmentCalculator 根据起止港拆分成航段
8. 查询 LOAD cargo_note 和 UNLOAD cargo_note
9. 开启数据库事务：
   a. GET_LOCK("voyage_<line>_<vessel>_<date>", 10)
   b. SELECT ... FOR UPDATE 锁定所有涉及航段
   c. CapacityChecker 校验容量
   d. 创建 shipping_order
   e. 批量创建 order_cargo
   f. 批量创建 segment_capacity_usage
   g. 更新 cargo_note 的 cumulative_booked_capacity_ton
   h. RELEASE_LOCK
10. 后置：缓存 DeletePrefix("voyage_rec:")
```

两层锁 + 事务保证了并发安全。

---

### Q43：Handler 层如何做权限校验？

登录时根据传入的 `role` 参数分流到不同的 service：

```go
switch req.Role {
case "shipper":
    company, err := h.shipperSvc.Login(...)
case "shipping":
    company, err := h.shippingSvc.Login(...)
case "admin":
    admin, err := h.adminSvc.Login(...)
}
```

中间件层两层校验：
- `RequireAuth()`：验证 JWT，从 token 中解析 `user_id`、`username`、`role`，注入 Gin context
- `RequireRole("admin")`：从 context 取 role，校验是否匹配。只有 `/admin/*` 路由加了这层

Handler 层还有业务级权限校验：shipper 只能查自己的订单（`user_id == shipper_company_id`）。

---

### Q44：退出流程是如何设计的？

```go
<-quit               // 1. 等待 SIGINT/SIGTERM 信号
ws.ShutdownHub()     // 2. 先关 WebSocket（不再推消息）
srv.Shutdown(ctx)    // 3. 再关 HTTP（不再接新请求，等老请求处理完）
```

先关 WebSocket 再关 HTTP。WebSocket 是服务端推送通道，关闭后用户不再收到通知，但 HTTP API 还能继续处理请求。如果反过来，用户刚下的订单收不到状态推送，体验差。

同时每 30 秒打一次数据库连接池状态日志（`open`、`in_use`、`idle`、`wait_count`），运维可以从日志趋势发现数据库问题。

---

### Q45：整个系统的完整目录结构如何？

```
backend/
├── app/main.go           ← 启动入口
├── cmd/                  ← CLI 命令
├── config.yaml           ← 配置文件
├── docs/                 ← Swagger 文档
├── excel/                ← Excel 模板
├── go.mod / go.sum       ← 依赖管理
├── internal/
│   ├── biz/              ← 纯业务逻辑
│   ├── dao/              ← 数据访问层
│   ├── handler/          ← HTTP 请求处理器
│   ├── model/            ← GORM 模型
│   ├── notify/           ← 通知提供者（邮件/SMS）
│   └── service/          ← 业务编排层
├── net/
│   ├── middleware/       ← HTTP 中间件
│   ├── protect/          ← 安全防护
│   ├── router/           ← 路由注册
│   └── websocket/        ← WebSocket
├── pkg/
│   ├── cache/            ← 内存缓存
│   ├── config/           ← 配置加载
│   ├── crypto/           ← 加密工具
│   ├── database/         ← 数据库连接
│   ├── errors/           ← 错误类型
│   ├── excel/            ← Excel 读写
│   ├── fileutil/         ← 文件工具
│   ├── idgen/            ← ID 生成器
│   ├── jwt/              ← JWT 认证
│   ├── logger/           ← 日志
│   ├── response/         ← 统一响应
│   ├── timeutil/         ← 时间工具
│   └── validator/        ← 参数校验
├── project_log/          ← 项目日志
├── sql/                  ← 数据库脚本
│   ├── tables_mysql.sql
│   ├── optimizations_mysql.sql
│   ├── SQL.md
│   └── E-R.png
└── QA.md                 ← 本文档
```

---

*本文档由后端代码和数据库脚本逐行分析整理而成，覆盖全部设计决策的"做了什么"和"为什么这么做"。*
