-- =============================================================================
-- Database Schema for Shipping Logistics Management System (MySQL)
-- Converted from Oracle version, preserving all designs.
-- All table names, column names follow lowercase with underscores.
-- No Chinese characters.
-- =============================================================================

CREATE DATABASE IF NOT EXISTS mts;
USE mts;

-- =============================================================================
-- 1. Parent Tables (no foreign dependencies)
-- =============================================================================

-- 1.1 city
CREATE TABLE city (
    city_id             BIGINT PRIMARY KEY AUTO_INCREMENT,
    city_name           VARCHAR(100) NOT NULL,
    country             VARCHAR(100),
    country_code        VARCHAR(10),
    timezone            VARCHAR(50),
    latitude            DECIMAL(10,6),
    longitude           DECIMAL(10,6),
    create_time         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time         DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time         DATETIME
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 1.2 shipper_company
CREATE TABLE shipper_company (
    company_id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
    company_name                VARCHAR(200) NOT NULL,
    unified_social_credit_code  VARCHAR(50),
    legal_representative        VARCHAR(100),
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

-- 1.3 shipping_company
CREATE TABLE shipping_company (
    company_id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
    company_name                VARCHAR(200) NOT NULL,
    unified_social_credit_code  VARCHAR(50),
    contact_person              VARCHAR(100),
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

-- 1.4 admin
CREATE TABLE admin (
    admin_id        BIGINT PRIMARY KEY AUTO_INCREMENT,
    username        VARCHAR(100) NOT NULL,
    password        VARCHAR(255) NOT NULL,
    real_name       VARCHAR(100),
    role            TINYINT DEFAULT 2,
    create_time     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time     DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time     DATETIME,
    UNIQUE INDEX uk_username_delete (username, delete_time)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 1.5 port (depends on city)
CREATE TABLE port (
    port_id             BIGINT PRIMARY KEY AUTO_INCREMENT,
    port_name           VARCHAR(200) NOT NULL,
    port_code           VARCHAR(50) UNIQUE,
    city_id             BIGINT,
    latitude            DECIMAL(10,6),
    longitude           DECIMAL(10,6),
    port_type           VARCHAR(50),
    max_draft_meter     DECIMAL(6,2),
    create_time         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time         DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time         DATETIME,
    CONSTRAINT fk_port_city FOREIGN KEY (city_id) REFERENCES city(city_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 1.6 berth (depends on port)
CREATE TABLE berth (
    berth_id                BIGINT PRIMARY KEY AUTO_INCREMENT,
    berth_name              VARCHAR(100) NOT NULL,
    port_id                 BIGINT,
    berth_type              VARCHAR(50),
    draft_meter             DECIMAL(6,2),
    length_meter            DECIMAL(8,2),
    width_meter             DECIMAL(8,2),
    max_berthing_tonnage    DECIMAL(12,2),
    functional_zone         VARCHAR(100),
    is_available            TINYINT DEFAULT 1,
    create_time             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time             DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time             DATETIME,
    CONSTRAINT fk_berth_port FOREIGN KEY (port_id) REFERENCES port(port_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 1.7 vessel (depends on shipping_company)
CREATE TABLE vessel (
    vessel_id               BIGINT PRIMARY KEY AUTO_INCREMENT,
    vessel_name             VARCHAR(200) NOT NULL,
    call_sign               VARCHAR(50),
    imo_number              VARCHAR(20) NOT NULL,
    vessel_type             VARCHAR(50),
    max_deadweight_ton      DECIMAL(12,2),
    gross_tonnage           DECIMAL(12,2),
    net_tonnage             DECIMAL(12,2),
    draft_meter             DECIMAL(6,2),
    speed_knot              DECIMAL(6,2),
    container_teu           INT,
    is_available            TINYINT DEFAULT 1,
    shipping_company_id     BIGINT,
    create_time             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time             DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time             DATETIME,
    CONSTRAINT fk_vessel_shipping_company FOREIGN KEY (shipping_company_id) REFERENCES shipping_company(company_id),
    UNIQUE INDEX uk_imo_delete (imo_number, delete_time)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 1.8 shipping_line (depends on shipping_company)
CREATE TABLE shipping_line (
    line_id                 BIGINT PRIMARY KEY AUTO_INCREMENT,
    line_name               VARCHAR(200) NOT NULL,
    shipping_company_id     BIGINT,
    port_sequence           JSON,
    total_distance_nm       DECIMAL(10,2),
    departure_port_name     VARCHAR(200),
    destination_port_name   VARCHAR(200),
    description             TEXT,
    create_time             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time             DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_time             DATETIME,
    CONSTRAINT fk_shipping_line_company FOREIGN KEY (shipping_company_id) REFERENCES shipping_company(company_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 1.9 voyage_cargo_note (depends on shipping_line and vessel)
CREATE TABLE voyage_cargo_note (
    note_id                     BIGINT PRIMARY KEY AUTO_INCREMENT,
    line_id                     BIGINT,
    vessel_id                   BIGINT,
    voyage_date                 DATE NOT NULL,
    sequence_no                 INT NOT NULL,
    cargo_name                  VARCHAR(200),
    cargo_type                  VARCHAR(50),
    quantity                    DECIMAL(18,2),
    weight_ton                  DECIMAL(18,3),
    volume_cubic_meter          DECIMAL(18,3),
    unit_price                  DECIMAL(18,2),
    subtotal                    DECIMAL(18,2),
    operation_type              VARCHAR(20),
    cargo_handled_ton           DECIMAL(18,3),
    cumulative_booked_capacity_ton DECIMAL(18,3),
    create_time                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time                 DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_cargonote_line FOREIGN KEY (line_id) REFERENCES shipping_line(line_id),
    CONSTRAINT fk_cargonote_vessel FOREIGN KEY (vessel_id) REFERENCES vessel(vessel_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 2.1 voyage_berthing (depends on shipping_line, vessel, port, berth)
CREATE TABLE voyage_berthing (
    berthing_id                 BIGINT PRIMARY KEY AUTO_INCREMENT,
    line_id                     BIGINT,
    vessel_id                   BIGINT,
    voyage_date                 DATE NOT NULL,
    sequence_no                 INT NOT NULL,
    port_id                     BIGINT,
    berth_id                    BIGINT,
    planned_arrival_time        DATETIME,
    planned_departure_time      DATETIME,
    actual_arrival_time         DATETIME,
    actual_departure_time       DATETIME,
    draft_at_berthing_meter     DECIMAL(6,2),
    is_adjustable               TINYINT DEFAULT 1,
    create_time                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time                 DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_berthing_line FOREIGN KEY (line_id) REFERENCES shipping_line(line_id),
    CONSTRAINT fk_berthing_vessel FOREIGN KEY (vessel_id) REFERENCES vessel(vessel_id),
    CONSTRAINT fk_berthing_port FOREIGN KEY (port_id) REFERENCES port(port_id),
    CONSTRAINT fk_berthing_berth FOREIGN KEY (berth_id) REFERENCES berth(berth_id),
    CONSTRAINT uk_berthing_line_vessel_date_seq UNIQUE (line_id, vessel_id, voyage_date, sequence_no)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- 2.2 shipping_order (depends on shipper_company, city, voyage_cargo_note, port)
CREATE TABLE shipping_order (
    order_id                    BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_no                    VARCHAR(50) NOT NULL,
    shipper_company_id          BIGINT,
    city_id                     BIGINT,
    load_note_id                BIGINT,
    unload_note_id              BIGINT,
    departure_port_id           BIGINT,
    destination_port_id         BIGINT,
    expected_departure_date     DATE,
    expected_arrival_date       DATE,
    total_cost                  DECIMAL(18,2),
    shipper_contact             VARCHAR(200),
    consignee_contact           VARCHAR(200),
    payment_status              TINYINT,
    order_status                TINYINT,
    total_weight_ton            DECIMAL(18,3),
    total_volume_cubic_meter    DECIMAL(18,3),
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

-- 2.3 order_cargo (depends on shipping_order)
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

-- 2.4 segment_capacity_usage (depends on shipping_order, shipping_line, vessel, port)
CREATE TABLE segment_capacity_usage (
    usage_id            BIGINT PRIMARY KEY AUTO_INCREMENT,
    order_id            BIGINT,
    line_id             BIGINT,
    vessel_id           BIGINT,
    voyage_date         DATE NOT NULL,
    start_port_id       BIGINT,
    end_port_id         BIGINT,
    occupied_ton        DECIMAL(18,3) NOT NULL,
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
-- 3. Indexes for Foreign Keys (Performance)
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

-- Additional business indexes
CREATE INDEX idx_order_shipper_status ON shipping_order(shipper_company_id, order_status);
CREATE INDEX idx_cargonote_line_date ON voyage_cargo_note(line_id, voyage_date);

-- =============================================================================
-- 4. Indexes for delete_time columns (soft delete queries)
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