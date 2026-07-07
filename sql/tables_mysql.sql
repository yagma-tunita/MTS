-- =============================================================================
-- MTS 数据库建表脚本（MySQL）
-- 从 Oracle 版本转换而来，保留所有设计。
-- 所有表名、字段名使用小写加下划线。
-- 设计原则：
--   1. 所有实体均支持软删除（delete_time 字段），查询时需 WHERE delete_time IS NULL
--   2. 唯一索引通常包含 delete_time 以实现软删除后的唯一性（允许重建同名记录）
--   3. 使用 InnoDB 引擎 + utf8mb4 字符集
-- =============================================================================

CREATE DATABASE IF NOT EXISTS mts;
USE mts;

-- =============================================================================
-- 1. 基础表（无外键依赖）
-- =============================================================================

-- 1.1 城市表：存储港口所在城市的基本信息
CREATE TABLE city (
    city_id             BIGINT PRIMARY KEY AUTO_INCREMENT,
    city_name           VARCHAR(100) NOT NULL,       -- 城市名
    country             VARCHAR(100),                 -- 所属国家
    country_code        VARCHAR(10),                  -- 国家代码（如 CN、SG）
    timezone            VARCHAR(50),                  -- 时区（如 Asia/Shanghai）
    latitude            DECIMAL(10,6),                -- 纬度
    longitude           DECIMAL(10,6),                -- 经度
    create_time         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time         DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time         DATETIME                      -- 软删除时间
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 1.2 货主公司表：托运货物的公司（shipper 角色）
CREATE TABLE shipper_company (
    company_id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
    company_name                VARCHAR(200) NOT NULL,  -- 公司名称
    unified_social_credit_code  VARCHAR(50),             -- 统一社会信用代码
    legal_representative        VARCHAR(100),            -- 法定代表人
    contact_phone               VARCHAR(50),             -- 联系电话
    address                     VARCHAR(500),            -- 地址
    login_username              VARCHAR(100) NOT NULL,   -- 登录用户名
    login_password              VARCHAR(255) NOT NULL,   -- bcrypt 哈希密码
    account_status              TINYINT DEFAULT 1,       -- 账号状态（1=启用，0=禁用）
    create_time                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time                 DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time                 DATETIME,
    UNIQUE INDEX uk_social_credit_delete (unified_social_credit_code, delete_time),
    UNIQUE INDEX uk_username_delete (login_username, delete_time)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 1.3 船公司表：提供运输服务的公司（shipping 角色）
CREATE TABLE shipping_company (
    company_id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
    company_name                VARCHAR(200) NOT NULL,
    unified_social_credit_code  VARCHAR(50),
    contact_person              VARCHAR(100),            -- 联系人
    contact_phone               VARCHAR(50),
    address                     VARCHAR(500),
    login_username              VARCHAR(100) NOT NULL,
    login_password              VARCHAR(255) NOT NULL,
    account_status              TINYINT DEFAULT 1,
    create_time                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time                 DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time                 DATETIME,
    UNIQUE INDEX uk_social_credit_delete (unified_social_credit_code, delete_time),
    UNIQUE INDEX uk_username_delete (login_username, delete_time)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 1.4 管理员表（admin 角色）
CREATE TABLE admin (
    admin_id        BIGINT PRIMARY KEY AUTO_INCREMENT,
    username        VARCHAR(100) NOT NULL,
    password        VARCHAR(255) NOT NULL,
    real_name       VARCHAR(100),
    role            TINYINT DEFAULT 2,     -- 角色（1=超级管理员，2=普通管理员）
    create_time     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time     DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time     DATETIME,
    UNIQUE INDEX uk_username_delete (username, delete_time)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 1.5 港口表：依赖城市
CREATE TABLE port (
    port_id             BIGINT PRIMARY KEY AUTO_INCREMENT,
    port_name           VARCHAR(200) NOT NULL,
    port_code           VARCHAR(50) UNIQUE,    -- 港口代码（如 CNSHA）
    city_id             BIGINT,                -- 所在城市
    latitude            DECIMAL(10,6),
    longitude           DECIMAL(10,6),
    port_type           VARCHAR(50),            -- 港口类型（Sea Port、River Port 等）
    max_draft_meter     DECIMAL(6,2),           -- 最大吃水深度（米）
    create_time         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time         DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time         DATETIME,
    CONSTRAINT fk_port_city FOREIGN KEY (city_id) REFERENCES city(city_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 1.6 泊位表：属于港口的具体泊位
CREATE TABLE berth (
    berth_id                BIGINT PRIMARY KEY AUTO_INCREMENT,
    berth_name              VARCHAR(100) NOT NULL,
    port_id                 BIGINT,
    berth_type              VARCHAR(50),          -- 泊位类型（Container、Bulk 等）
    draft_meter             DECIMAL(6,2),         -- 水深（米）
    length_meter            DECIMAL(8,2),         -- 长度（米）
    width_meter             DECIMAL(8,2),         -- 宽度（米）
    max_berthing_tonnage    DECIMAL(12,2),        -- 最大靠泊吨位（吨）
    functional_zone         VARCHAR(100),         -- 功能区
    is_available            TINYINT DEFAULT 1,    -- 是否可用
    create_time             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time             DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time             DATETIME,
    CONSTRAINT fk_berth_port FOREIGN KEY (port_id) REFERENCES port(port_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 1.7 船舶表：依赖船公司
CREATE TABLE vessel (
    vessel_id               BIGINT PRIMARY KEY AUTO_INCREMENT,
    vessel_name             VARCHAR(200) NOT NULL,
    call_sign               VARCHAR(50),          -- 呼号
    imo_number              VARCHAR(20) NOT NULL, -- IMO 编号（唯一标识）
    vessel_type             VARCHAR(50),          -- 船舶类型（Container Ship、Bulk Carrier 等）
    max_deadweight_ton      DECIMAL(12,2),        -- 最大载重吨（DWT）
    gross_tonnage           DECIMAL(12,2),        -- 总吨位
    net_tonnage             DECIMAL(12,2),        -- 净吨位
    draft_meter             DECIMAL(6,2),         -- 吃水深度
    speed_knot              DECIMAL(6,2),         -- 航速（节）
    container_teu           INT,                  -- 集装箱容量（TEU）
    is_available            TINYINT DEFAULT 1,    -- 是否可用
    shipping_company_id     BIGINT,               -- 所属船公司
    create_time             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time             DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time             DATETIME,
    CONSTRAINT fk_vessel_shipping_company FOREIGN KEY (shipping_company_id) REFERENCES shipping_company(company_id),
    UNIQUE INDEX uk_imo_delete (imo_number, delete_time)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 1.8 航线表：由船公司定义的固定航线，含港口序列（JSON 格式）
CREATE TABLE shipping_line (
    line_id                 BIGINT PRIMARY KEY AUTO_INCREMENT,
    line_name               VARCHAR(200) NOT NULL,
    shipping_company_id     BIGINT,               -- 运营船公司
    port_sequence           JSON,                  -- 港口序列表，如 [1, 2, 3, 5, 7]（JSON 数组）
    total_distance_nm       DECIMAL(10,2),        -- 总航程（海里）
    departure_port_name     VARCHAR(200),          -- 出发港名称（冗余）
    destination_port_name   VARCHAR(200),          -- 目的港名称（冗余）
    description             TEXT,                  -- 航线描述
    create_time             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time             DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time             DATETIME,
    CONSTRAINT fk_shipping_line_company FOREIGN KEY (shipping_company_id) REFERENCES shipping_company(company_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 1.9 航次货物记录表：记录每个航次在各港口的装/卸货计划
CREATE TABLE voyage_cargo_note (
    note_id                     BIGINT PRIMARY KEY AUTO_INCREMENT,
    line_id                     BIGINT,
    vessel_id                   BIGINT,
    voyage_date                 DATE NOT NULL,        -- 航次日期
    sequence_no                 INT NOT NULL,         -- 港口序号（对应 port_sequence 中的位置）
    cargo_name                  VARCHAR(200),         -- 货物名称
    cargo_type                  VARCHAR(50),          -- 货物类型（bulk/container/liquid）
    quantity                    DECIMAL(18,2),        -- 数量
    weight_ton                  DECIMAL(18,3),        -- 重量（吨）
    volume_cubic_meter          DECIMAL(18,3),        -- 体积（立方米）
    unit_price                  DECIMAL(18,2),        -- 单价
    subtotal                    DECIMAL(18,2),        -- 小计
    operation_type              VARCHAR(20),          -- 操作类型（LOAD=装货，UNLOAD=卸货）
    cargo_handled_ton           DECIMAL(18,3),        -- 已操作吨数
    cumulative_booked_capacity_ton DECIMAL(18,3),     -- 累积已预订容量（订单创建时累加）
    create_time                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time                 DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_cargonote_line FOREIGN KEY (line_id) REFERENCES shipping_line(line_id),
    CONSTRAINT fk_cargonote_vessel FOREIGN KEY (vessel_id) REFERENCES vessel(vessel_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- =============================================================================
-- 2. 业务表（有外键依赖）
-- =============================================================================

-- 2.1 航次靠泊表：记录每个航次在各港口的靠泊计划与实际时间
CREATE TABLE voyage_berthing (
    berthing_id                 BIGINT PRIMARY KEY AUTO_INCREMENT,
    line_id                     BIGINT,
    vessel_id                   BIGINT,
    voyage_date                 DATE NOT NULL,
    sequence_no                 INT NOT NULL,             -- 靠泊顺序
    port_id                     BIGINT,                   -- 靠泊港口
    berth_id                    BIGINT,                   -- 靠泊泊位
    planned_arrival_time        DATETIME,                  -- 计划到达时间
    planned_departure_time      DATETIME,                  -- 计划离港时间
    actual_arrival_time         DATETIME,                  -- 实际到达时间
    actual_departure_time       DATETIME,                  -- 实际离港时间
    draft_at_berthing_meter     DECIMAL(6,2),              -- 靠泊吃水
    is_adjustable               TINYINT DEFAULT 1,         -- 是否可调整
    create_time                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time                 DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_berthing_line FOREIGN KEY (line_id) REFERENCES shipping_line(line_id),
    CONSTRAINT fk_berthing_vessel FOREIGN KEY (vessel_id) REFERENCES vessel(vessel_id),
    CONSTRAINT fk_berthing_port FOREIGN KEY (port_id) REFERENCES port(port_id),
    CONSTRAINT fk_berthing_berth FOREIGN KEY (berth_id) REFERENCES berth(berth_id),
    CONSTRAINT uk_berthing_line_vessel_date_seq UNIQUE (line_id, vessel_id, voyage_date, sequence_no)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 2.2 运输订单表：核心业务表，记录货物的运输订单
CREATE TABLE shipping_order (
    order_id                    BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_no                    VARCHAR(50) NOT NULL,       -- 订单号（ORD20260706xxxxxx）
    shipper_company_id          BIGINT,                     -- 货主公司
    city_id                     BIGINT,                     -- 城市
    load_note_id                BIGINT,                     -- 装货 cargo note
    unload_note_id              BIGINT,                     -- 卸货 cargo note
    departure_port_id           BIGINT,                     -- 出发港
    destination_port_id         BIGINT,                     -- 目的港
    expected_departure_date     DATE,                       -- 预计离港日期
    expected_arrival_date       DATE,                       -- 预计到达日期
    total_cost                  DECIMAL(18,2),              -- 总运费
    shipper_contact             VARCHAR(200),               -- 货主联系方式
    consignee_contact           VARCHAR(200),               -- 收货人联系方式
    payment_status              TINYINT,                    -- 支付状态（0=未支付，1=已支付）
    order_status                TINYINT,                    -- 订单状态（0=草稿，1=已确认，2=运输中，3=已完成，4=已取消）
    total_weight_ton            DECIMAL(18,3),              -- 总重量（吨）
    total_volume_cubic_meter    DECIMAL(18,3),              -- 总体积（立方米）
    create_time                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time                 DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time                 DATETIME,
    CONSTRAINT fk_order_shipper FOREIGN KEY (shipper_company_id) REFERENCES shipper_company(company_id),
    CONSTRAINT fk_order_city FOREIGN KEY (city_id) REFERENCES city(city_id),
    CONSTRAINT fk_order_load_note FOREIGN KEY (load_note_id) REFERENCES voyage_cargo_note(note_id),
    CONSTRAINT fk_order_unload_note FOREIGN KEY (unload_note_id) REFERENCES voyage_cargo_note(note_id),
    CONSTRAINT fk_order_departure_port FOREIGN KEY (departure_port_id) REFERENCES port(port_id),
    CONSTRAINT fk_order_destination_port FOREIGN KEY (destination_port_id) REFERENCES port(port_id),
    UNIQUE INDEX uk_orderno_delete (order_no, delete_time)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 2.3 订单货物明细表：每个订单包含的货物条目
CREATE TABLE order_cargo (
    detail_id               BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_id                BIGINT,
    cargo_name              VARCHAR(200),
    cargo_type              VARCHAR(50),
    quantity                DECIMAL(18,2),
    weight_ton              DECIMAL(18,3),
    volume_cubic_meter      DECIMAL(18,3),
    unit_price              DECIMAL(18,2),
    subtotal                DECIMAL(18,2),
    create_time             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time             DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time             DATETIME,
    CONSTRAINT fk_cargo_order FOREIGN KEY (order_id) REFERENCES shipping_order(order_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 2.4 航段容量占用表：记录每个订单在航线各航段占用的吨位，用于容量计算
CREATE TABLE segment_capacity_usage (
    usage_id            BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_id            BIGINT,
    line_id             BIGINT,
    vessel_id           BIGINT,
    voyage_date         DATE NOT NULL,
    start_port_id       BIGINT,                        -- 航段起始港口
    end_port_id         BIGINT,                        -- 航段结束港口
    occupied_ton        DECIMAL(18,3) NOT NULL,        -- 占用的吨位
    create_time         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_usage_order FOREIGN KEY (order_id) REFERENCES shipping_order(order_id),
    CONSTRAINT fk_usage_line FOREIGN KEY (line_id) REFERENCES shipping_line(line_id),
    CONSTRAINT fk_usage_vessel FOREIGN KEY (vessel_id) REFERENCES vessel(vessel_id),
    CONSTRAINT fk_usage_start_port FOREIGN KEY (start_port_id) REFERENCES port(port_id),
    CONSTRAINT fk_usage_end_port FOREIGN KEY (end_port_id) REFERENCES port(port_id),
    UNIQUE INDEX uk_usage_unique (order_id, line_id, vessel_id, voyage_date, start_port_id, end_port_id),
    INDEX idx_usage_query (line_id, vessel_id, voyage_date, start_port_id, end_port_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- =============================================================================
-- 3. 外键索引（提升 JOIN 查询性能）
-- =============================================================================

CREATE INDEX idx_port_city_id ON port(city_id);
CREATE INDEX idx_berth_port_id ON berth(port_id);
CREATE INDEX idx_vessel_shipping_company_id ON vessel(shipping_company_id);
CREATE INDEX idx_shipping_line_company_id ON shipping_line(shipping_company_id);
CREATE INDEX idx_cargonote_line_id ON voyage_cargo_note(line_id);
CREATE INDEX idx_cargonote_vessel_id ON voyage_cargo_note(vessel_id);
CREATE INDEX idx_berthing_line_id ON voyage_berthing(line_id);
CREATE INDEX idx_berthing_vessel_id ON voyage_berthing(vessel_id);
CREATE INDEX idx_berthing_port_id ON voyage_berthing(port_id);
CREATE INDEX idx_berthing_berth_id ON voyage_berthing(berth_id);
CREATE INDEX idx_order_shipper_company_id ON shipping_order(shipper_company_id);
CREATE INDEX idx_order_city_id ON shipping_order(city_id);
CREATE INDEX idx_order_load_note_id ON shipping_order(load_note_id);
CREATE INDEX idx_order_unload_note_id ON shipping_order(unload_note_id);
CREATE INDEX idx_order_departure_port_id ON shipping_order(departure_port_id);
CREATE INDEX idx_order_destination_port_id ON shipping_order(destination_port_id);
CREATE INDEX idx_cargo_order_id ON order_cargo(order_id);
CREATE INDEX idx_usage_order_id ON segment_capacity_usage(order_id);
CREATE INDEX idx_usage_line_id ON segment_capacity_usage(line_id);
CREATE INDEX idx_usage_vessel_id ON segment_capacity_usage(vessel_id);
CREATE INDEX idx_usage_start_port_id ON segment_capacity_usage(start_port_id);
CREATE INDEX idx_usage_end_port_id ON segment_capacity_usage(end_port_id);

-- 业务查询常用组合索引
CREATE INDEX idx_order_shipper_status ON shipping_order(shipper_company_id, order_status);
CREATE INDEX idx_cargonote_line_date ON voyage_cargo_note(line_id, voyage_date);

-- =============================================================================
-- 4. 软删除索引（WHERE delete_time IS NULL 查询优化）
-- =============================================================================

CREATE INDEX idx_city_delete_time ON city(delete_time);
CREATE INDEX idx_shipper_company_delete_time ON shipper_company(delete_time);
CREATE INDEX idx_shipping_company_delete_time ON shipping_company(delete_time);
CREATE INDEX idx_admin_delete_time ON admin(delete_time);
CREATE INDEX idx_port_delete_time ON port(delete_time);
CREATE INDEX idx_berth_delete_time ON berth(delete_time);
CREATE INDEX idx_vessel_delete_time ON vessel(delete_time);
CREATE INDEX idx_shipping_line_delete_time ON shipping_line(delete_time);
CREATE INDEX idx_shipping_order_delete_time ON shipping_order(delete_time);
-- =============================================================================
-- End of script
-- =============================================================================
