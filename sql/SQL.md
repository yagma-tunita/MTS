===============================================================================
                        数据库设计说明文档
===============================================================================

1. 概述

本数据库服务于航运物流管理系统，涵盖货主、船运公司、管理员、城市、港口、泊位、
船只、航线、航次靠泊、航次货单、订单、订单货物、航段载重等核心业务实体。
设计遵循第三范式（3NF），支持动态货运管理，包括中途卸货后释放运力、实时载重校验等业务。

所有表名、字段名均采用英文小写+下划线命名风格，符合主流数据库命名规范。

===============================================================================
2. 表结构定义
===============================================================================

2.1 货主公司 (shipper_company)
-------------------------------------------------------------------------------
字段名                        类型          约束                        说明
company_id                    bigint        PRIMARY KEY                 公司编号
company_name                  varchar       NOT NULL                    公司名称
unified_social_credit_code    varchar       UNIQUE                      统一社会信用代码
legal_representative          varchar                                  法定代表人
contact_phone                 varchar                                  联系电话
address                       varchar                                  地址
login_username                varchar       UNIQUE NOT NULL             登录用户名
login_password                varchar       NOT NULL                    登录密码（存储哈希）
account_status                tinyint       DEFAULT 1                   账号状态（1启用，0停用）
create_time                   datetime      NOT NULL                    创建时间
update_time                   datetime                                  更新时间
delete_time                   datetime                                  删除时间（软删除）


2.2 船运公司 (shipping_company)
-------------------------------------------------------------------------------
字段名                        类型          约束                        说明
company_id                    bigint        PRIMARY KEY                 公司编号
company_name                  varchar       NOT NULL                    公司名称
unified_social_credit_code    varchar       UNIQUE                      统一社会信用代码
contact_person                varchar                                  联系人
contact_phone                 varchar                                  联系电话
address                       varchar                                  地址
login_username                varchar       UNIQUE NOT NULL             登录用户名
login_password                varchar       NOT NULL                    登录密码（存储哈希）
account_status                tinyint       DEFAULT 1                   账号状态（1启用，0停用）
create_time                   datetime      NOT NULL                    创建时间
update_time                   datetime                                  更新时间
delete_time                   datetime                                  删除时间


2.3 管理员 (admin)
-------------------------------------------------------------------------------
字段名                        类型          约束                        说明
admin_id                      bigint        PRIMARY KEY                 管理员编号
username                      varchar       UNIQUE NOT NULL             用户名
password                      varchar       NOT NULL                    密码（存储哈希）
real_name                     varchar                                  姓名
role                          tinyint       DEFAULT 2                   角色（1超级管理员，2普通管理员）
create_time                   datetime      NOT NULL                    创建时间
update_time                   datetime                                  更新时间


2.4 城市 (city)
-------------------------------------------------------------------------------
字段名                        类型          约束                        说明
city_id                       bigint        PRIMARY KEY                 城市编号
city_name                     varchar       NOT NULL                    城市名称
country                       varchar                                  国家
country_code                  varchar                                  国家代码（如CN，US）
timezone                      varchar                                  时区（如Asia/Shanghai）
latitude                      decimal(10,6)                            纬度
longitude                     decimal(10,6)                            经度
create_time                   datetime      NOT NULL                    创建时间
update_time                   datetime                                  更新时间
delete_time                   datetime                                  删除时间


2.5 港口 (port)
-------------------------------------------------------------------------------
字段名                        类型          约束                        说明
port_id                       bigint        PRIMARY KEY                 港口编号
port_name                     varchar       NOT NULL                    港口名称
port_code                     varchar       UNIQUE                      港口代码（如联合国口岸代码）
city_id                       bigint        FOREIGN KEY → city(city_id)  城市编号
latitude                      decimal(10,6)                            纬度
longitude                     decimal(10,6)                            经度
port_type                     varchar                                  港口类型（海港、河港等）
max_draft_meter               decimal(6,2)                             最大吃水（米）
create_time                   datetime      NOT NULL                    创建时间
update_time                   datetime                                  更新时间
delete_time                   datetime                                  删除时间


2.6 泊位 (berth)
-------------------------------------------------------------------------------
字段名                        类型          约束                        说明
berth_id                      bigint        PRIMARY KEY                 泊位编号
berth_name                    varchar       NOT NULL                    泊位名称
port_id                       bigint        FOREIGN KEY → port(port_id)  港口编号
berth_type                    varchar                                  泊位类型（散货、集装箱、油品等）
draft_meter                   decimal(6,2)                             吃水深度（米）
length_meter                  decimal(8,2)                             长度（米）
width_meter                   decimal(8,2)                             宽度（米）
max_berthing_tonnage          decimal(12,2)                            最大靠泊吨位
functional_zone               varchar                                  功能分区
is_available                  tinyint       DEFAULT 1                  是否可用（0-不可用，1-可用）
create_time                   datetime      NOT NULL                    创建时间
update_time                   datetime                                  更新时间
delete_time                   datetime                                  删除时间


2.7 船只 (vessel)
-------------------------------------------------------------------------------
字段名                        类型          约束                        说明
vessel_id                     bigint        PRIMARY KEY                 船只编号
vessel_name                   varchar       NOT NULL                    船名
call_sign                     varchar                                  船舶呼号
imo_number                    varchar       UNIQUE NOT NULL             IMO编号（国际海事组织唯一标识）
vessel_type                   varchar                                  船型（如集装箱船、散货船等）
max_deadweight_ton            decimal(12,2)                            最大载重（吨）
gross_tonnage                 decimal(12,2)                            总吨位
net_tonnage                   decimal(12,2)                            净吨位
draft_meter                   decimal(6,2)                             吃水深度（米）
speed_knot                    decimal(6,2)                             航速（节）
container_teu                 decimal(8,0)                             集装箱容量（TEU）
is_available                  tinyint       DEFAULT 1                  是否可用（0-不可用，1-可用）
shipping_company_id           bigint        FOREIGN KEY → shipping_company(company_id)  船运公司编号
create_time                   datetime      NOT NULL                    创建时间
update_time                   datetime                                  更新时间
delete_time                   datetime                                  删除时间


2.8 航线 (shipping_line)
-------------------------------------------------------------------------------
字段名                        类型          约束                        说明
line_id                       bigint        PRIMARY KEY                 航线编号
line_name                     varchar       NOT NULL                    航线名称
shipping_company_id           bigint        FOREIGN KEY → shipping_company(company_id)  船运公司编号（管理该航线）
port_sequence                 text                                     途径港口序列（JSON或逗号分隔的港口ID，按顺序）
total_distance_nm             decimal(10,2)                            总距离（海里）
departure_port_name           varchar                                  起运港名称（冗余，由port_sequence第一个港口推导）
destination_port_name         varchar                                  目的港名称（冗余，由port_sequence最后一个港口推导）
description                   text                                     描述
create_time                   datetime      NOT NULL                    创建时间
update_time                   datetime                                  更新时间
delete_time                   datetime                                  删除时间


2.9 航次靠泊 (voyage_berthing)
-------------------------------------------------------------------------------
-- 记录每个航次（航线+船只+日期）在每个停靠港的泊位及计划/实际时间
字段名                        类型          约束                        说明
berthing_id                   bigint        PRIMARY KEY                 靠泊编号
line_id                       bigint        FOREIGN KEY → shipping_line(line_id)  航线编号
vessel_id                     bigint        FOREIGN KEY → vessel(vessel_id)        船只编号
voyage_date                   date          NOT NULL                    航次日期
sequence_no                   int           NOT NULL                    顺序号（同一航次内从1开始递增）
port_id                       bigint        FOREIGN KEY → port(port_id)            港口编号
berth_id                      bigint        FOREIGN KEY → berth(berth_id)          泊位编号
planned_arrival_time          datetime                                 计划到达时间
planned_departure_time        datetime                                 计划离开时间
actual_arrival_time           datetime                                 实际到达时间
actual_departure_time         datetime                                 实际离开时间
draft_at_berthing_meter       decimal(6,2)                             靠泊时船舶吃水（米）
is_adjustable                 boolean       DEFAULT TRUE               是否可调整
create_time                   datetime      NOT NULL                    创建时间
update_time                   datetime                                  更新时间
-- 唯一约束 (line_id, vessel_id, voyage_date, sequence_no)


2.10 航次货单 (voyage_cargo_note)
-------------------------------------------------------------------------------
-- 记录每个航次在每个停靠点的货物装卸明细
字段名                        类型          约束                        说明
note_id                       bigint        PRIMARY KEY                 货单编号
line_id                       bigint        FOREIGN KEY → shipping_line(line_id)  航线编号
vessel_id                     bigint        FOREIGN KEY → vessel(vessel_id)        船只编号
voyage_date                   date          NOT NULL                    航次日期
sequence_no                   int           NOT NULL                    顺序号（对应航次靠泊的停靠点）
cargo_name                    varchar                                  货物名称
cargo_type                    varchar                                  货物类型
quantity                      decimal(18,2)                            数量
weight_ton                    decimal(18,3)                            重量（吨）
volume_cubic_meter            decimal(18,3)                            体积（立方米）
unit_price                    decimal(18,2)                            单价
subtotal                      decimal(18,2)                            小计
operation_type                varchar                                  作业类型（LOAD装货，UNLOAD卸货）
cargo_handled_ton             decimal(18,3)                            装卸货量（吨）
cumulative_booked_capacity_ton decimal(18,3)                          离港时累计已订吨位
create_time                   datetime      NOT NULL                    创建时间
update_time                   datetime                                  更新时间


2.11 订单 (shipping_order)
-------------------------------------------------------------------------------
字段名                        类型          约束                        说明
order_id                      bigint        PRIMARY KEY                 订单编号
order_no                      varchar       UNIQUE NOT NULL             订单号（业务唯一标识）
shipper_company_id            bigint        FOREIGN KEY → shipper_company(company_id)  货主公司编号
city_id                       bigint        FOREIGN KEY → city(city_id)              城市编号（订单关联城市）
load_note_id                  bigint        FOREIGN KEY → voyage_cargo_note(note_id)  装货货单编号
unload_note_id                bigint        FOREIGN KEY → voyage_cargo_note(note_id)  卸货货单编号
departure_port_id             bigint        FOREIGN KEY → port(port_id)              起运港口编号
destination_port_id           bigint        FOREIGN KEY → port(port_id)              目的港口编号
expected_departure_date       date                                     期望起运日期
expected_arrival_date         date                                     期望到达日期
total_cost                    decimal(18,2)                            总费用
shipper_contact               varchar                                  发货联系人
consignee_contact             varchar                                  收货联系人
payment_status                tinyint                                  支付状态（0-未支付，1-已支付，2-部分支付）
order_status                  tinyint                                  订单状态（0-草稿，1-已确认，2-运输中，3-已完成，4-已取消）
total_weight_ton              decimal(18,3)                            货物总重量（吨）
total_volume_cubic_meter      decimal(18,3)                            货物总体积（立方米）
create_time                   datetime      NOT NULL                    创建时间
update_time                   datetime                                  更新时间
delete_time                   datetime                                  删除时间


2.12 订单货物 (order_cargo)
-------------------------------------------------------------------------------
字段名                        类型          约束                        说明
detail_id                     bigint        PRIMARY KEY                 明细编号
order_id                      bigint        FOREIGN KEY → shipping_order(order_id)  订单编号
cargo_name                    varchar                                  货物名称
cargo_type                    varchar                                  货物类型
quantity                      decimal(18,2)                            数量
weight_ton                    decimal(18,3)                            重量（吨）
volume_cubic_meter            decimal(18,3)                            体积（立方米）
unit_price                    decimal(18,2)                            单价
subtotal                      decimal(18,2)                            小计
create_time                   datetime      NOT NULL                    创建时间
update_time                   datetime                                  更新时间
delete_time                   datetime                                  删除时间


2.13 航段载重 (segment_capacity_usage)
-------------------------------------------------------------------------------
-- 记录每个订单占用的具体航段（起始港→目的港）及占用吨数，用于实时载重校验
字段名                        类型          约束                        说明
usage_id                      bigint        PRIMARY KEY                 载重编号
order_id                      bigint        FOREIGN KEY → shipping_order(order_id)  订单编号
line_id                       bigint        FOREIGN KEY → shipping_line(line_id)  航线编号
vessel_id                     bigint        FOREIGN KEY → vessel(vessel_id)        船只编号
voyage_date                   date          NOT NULL                    航次日期
start_port_id                 bigint        FOREIGN KEY → port(port_id)            起始港口编号
end_port_id                   bigint        FOREIGN KEY → port(port_id)            目的港口编号
occupied_ton                  decimal(18,3) NOT NULL                    占用吨数
create_time                   datetime      NOT NULL                    创建时间

===============================================================================
3. 关系说明（ER 关系）
===============================================================================

- 城市 (city) 1 ──┐ 位于 ┌── 港口 (port)
- 城市 (city) 1 ──┐ 关联 ┌── 订单 (shipping_order)
- 港口 (port) 1 ──┐ 拥有 ┌── 泊位 (berth)
- 港口 (port) 1 ──┐ 到访 ┌── 航次靠泊 (voyage_berthing)
- 泊位 (berth) 1 ──┐ 占用 ┌── 航次靠泊 (voyage_berthing)
- 船运公司 (shipping_company) 1 ──┐ 拥有 ┌── 船只 (vessel)
- 船运公司 (shipping_company) 1 ──┐ 管理 ┌── 航线 (shipping_line)
- 航线 (shipping_line) 1 ──┐ 生成 ┌── 航次靠泊 (voyage_berthing)
- 航线 (shipping_line) 1 ──┐ 生成 ┌── 航次货单 (voyage_cargo_note)
- 船只 (vessel) 1 ──┐ 执行 ┌── 航次靠泊 (voyage_berthing)
- 船只 (vessel) 1 ──┐ 执行 ┌── 航次货单 (voyage_cargo_note)
- 航次靠泊 (voyage_berthing) 1 ──┐ 关联 ┌── 航次货单 (voyage_cargo_note)  （通过 line_id, vessel_id, voyage_date, sequence_no）
- 货主公司 (shipper_company) 1 ──┐ 下单 ┌── 订单 (shipping_order)
- 订单 (shipping_order) 1 ──┐ 包含 ┌── 订单货物 (order_cargo)
- 订单 (shipping_order) 1 ──┐ 产生 ┌── 航段载重 (segment_capacity_usage)
- 航次货单 (voyage_cargo_note) 1 ──┐ 装货点 ┌── 订单 (shipping_order)
- 航次货单 (voyage_cargo_note) 1 ──┐ 卸货点 ┌── 订单 (shipping_order)

===============================================================================
4. 注意事项
===============================================================================

- 所有表均采用软删除（delete_time 字段），删除时仅标记时间，不物理删除记录。
- 航次通过 (line_id, vessel_id, voyage_date) 组合唯一标识，不使用独立航次表。
- 航次靠泊与航次货单通过 (line_id, vessel_id, voyage_date, sequence_no) 逻辑关联，无物理外键，由业务保证一致性。
- 订单中的 load_note_id 和 unload_note_id 引用航次货单，分别表示装货点和卸货点。
- 航段载重表用于新订单提交前快速校验剩余容量：查询同一航次下与起止港重叠的航段占用吨数之和，与船只最大载重吨比较。
- 航线表中的 port_sequence 存储静态途径港口顺序（如 [1001,1005,1008]），departure_port_name 和 destination_port_name 为冗余字段，便于快速展示。
- 所有密码字段（login_password, password）应存储哈希值，禁止明文存储。

===============================================================================
文档结束
===============================================================================