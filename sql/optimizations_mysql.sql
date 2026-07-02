-- =============================================================================
-- MTS Database Optimizations: Views, Stored Procedures, Triggers
-- This file does NOT modify the existing table DDL.
-- Run after tables_mysql.sql has been executed.
-- =============================================================================
-- Usage: mysql -u root -p mts < sql/optimizations_mysql.sql
-- =============================================================================

-- =============================================================================
-- 1. VIEWS — Simplify complex queries
-- =============================================================================

-- 1.1 vw_order_detail — 订单完整详情（含货物、港口、船舶、航线）
CREATE OR REPLACE VIEW vw_order_detail AS
SELECT
    so.order_id,
    so.order_no,
    so.shipper_company_id,
    sc.company_name AS shipper_company_name,
    so.city_id,
    so.departure_port_id,
    dp.port_name AS departure_port_name,
    so.destination_port_id,
    des.port_name AS destination_port_name,
    so.load_note_id,
    so.unload_note_id,
    so.expected_departure_date,
    so.expected_arrival_date,
    so.total_cost,
    so.payment_status,
    so.order_status,
    so.total_weight_ton,
    so.total_volume_cubic_meter,
    so.shipper_contact,
    so.consignee_contact,
    so.create_time AS order_create_time,
    ln.line_id,
    ln.line_name,
    v.vessel_id,
    v.vessel_name,
    v.imo_number
FROM shipping_order so
LEFT JOIN shipper_company sc ON sc.company_id = so.shipper_company_id
LEFT JOIN port dp ON dp.port_id = so.departure_port_id
LEFT JOIN port des ON des.port_id = so.destination_port_id
LEFT JOIN voyage_cargo_note lcn ON lcn.note_id = so.load_note_id
LEFT JOIN voyage_cargo_note ucn ON ucn.note_id = so.unload_note_id
LEFT JOIN shipping_line ln ON ln.line_id = lcn.line_id
LEFT JOIN vessel v ON v.vessel_id = lcn.vessel_id
WHERE so.delete_time IS NULL;

-- 1.2 vw_voyage_capacity — 航次各航段剩余容量
CREATE OR REPLACE VIEW vw_voyage_capacity AS
SELECT
    vb.line_id,
    vb.vessel_id,
    vb.voyage_date,
    vb.sequence_no,
    vb.port_id,
    p.port_name,
    vb.berth_id,
    v.max_deadweight_ton,
    COALESCE((
        SELECT SUM(scu.occupied_ton)
        FROM segment_capacity_usage scu
        WHERE scu.line_id = vb.line_id
          AND scu.vessel_id = vb.vessel_id
          AND scu.voyage_date = vb.voyage_date
          AND scu.start_port_id = vb.port_id
    ), 0) AS loaded_ton,
    v.max_deadweight_ton - COALESCE((
        SELECT SUM(scu.occupied_ton)
        FROM segment_capacity_usage scu
        WHERE scu.line_id = vb.line_id
          AND scu.vessel_id = vb.vessel_id
          AND scu.voyage_date = vb.voyage_date
          AND scu.start_port_id = vb.port_id
    ), 0) AS remaining_capacity_ton
FROM voyage_berthing vb
JOIN port p ON p.port_id = vb.port_id
JOIN vessel v ON v.vessel_id = vb.vessel_id;

-- 1.3 vw_port_sequence_detail — 航线港口序列详情（JSON 展开）
CREATE OR REPLACE VIEW vw_port_sequence_detail AS
SELECT
    sl.line_id,
    sl.line_name,
    JSON_LENGTH(sl.port_sequence) AS port_count,
    sl.total_distance_nm,
    sl.departure_port_name,
    sl.destination_port_name
FROM shipping_line sl
WHERE sl.delete_time IS NULL;


-- =============================================================================
-- 2. STORED PROCEDURES — Encapsulate complex business logic
-- =============================================================================

-- 2.1 sp_get_voyage_remaining_capacity — 获取航次指定航段的剩余容量
DELIMITER //

CREATE OR REPLACE PROCEDURE sp_get_voyage_remaining_capacity(
    IN p_line_id BIGINT,
    IN p_vessel_id BIGINT,
    IN p_voyage_date DATE,
    IN p_start_port_id BIGINT,
    IN p_end_port_id BIGINT,
    OUT remaining_ton DECIMAL(18,3)
)
BEGIN
    DECLARE used_ton DECIMAL(18,3);
    DECLARE max_ton DECIMAL(12,2);

    SELECT COALESCE(SUM(occupied_ton), 0) INTO used_ton
    FROM segment_capacity_usage
    WHERE line_id = p_line_id
      AND vessel_id = p_vessel_id
      AND voyage_date = p_voyage_date
      AND start_port_id >= p_start_port_id
      AND end_port_id <= p_end_port_id;

    SELECT max_deadweight_ton INTO max_ton
    FROM vessel
    WHERE vessel_id = p_vessel_id;

    SET remaining_ton = COALESCE(max_ton, 0) - used_ton;
END //

DELIMITER ;

-- 2.2 sp_recommend_voyages — 航次推荐（端口+需求吨位）
DELIMITER //

CREATE OR REPLACE PROCEDURE sp_recommend_voyages(
    IN p_start_port_id BIGINT,
    IN p_end_port_id BIGINT,
    IN p_required_ton DECIMAL(18,3)
)
BEGIN
    SELECT
        sl.line_id,
        sl.line_name,
        v.vessel_id,
        v.vessel_name,
        v.max_deadweight_ton,
        vcn.voyage_date,
        COALESCE(SUM(scu.occupied_ton), 0) AS total_occupied,
        v.max_deadweight_ton - COALESCE(SUM(scu.occupied_ton), 0) AS remaining_capacity
    FROM shipping_line sl
    JOIN vessel v ON v.shipping_company_id = sl.shipping_company_id
    JOIN voyage_cargo_note vcn ON vcn.line_id = sl.line_id AND vcn.vessel_id = v.vessel_id
    LEFT JOIN segment_capacity_usage scu ON scu.line_id = sl.line_id
        AND scu.vessel_id = v.vessel_id
        AND scu.voyage_date = vcn.voyage_date
        AND scu.start_port_id >= p_start_port_id
        AND scu.end_port_id <= p_end_port_id
    WHERE sl.delete_time IS NULL
      AND v.is_available = 1
      AND JSON_CONTAINS(sl.port_sequence, CAST(p_start_port_id AS JSON), '$')
      AND JSON_CONTAINS(sl.port_sequence, CAST(p_end_port_id AS JSON), '$')
      AND JSON_UNQUOTE(JSON_EXTRACT(sl.port_sequence, CONCAT('$[', JSON_UNQUERY(
          JSON_SEARCH(sl.port_sequence, 'one', CAST(p_start_port_id AS CHAR))
      ), ']'))) < JSON_UNQUOTE(JSON_EXTRACT(sl.port_sequence, CONCAT('$[', JSON_UNQUERY(
          JSON_SEARCH(sl.port_sequence, 'one', CAST(p_end_port_id AS CHAR))
      ), ']')))
    GROUP BY sl.line_id, v.vessel_id, vcn.voyage_date
    HAVING remaining_capacity >= p_required_ton
    ORDER BY remaining_capacity DESC;
END //

DELIMITER ;


-- =============================================================================
-- 3. TRIGGERS — Automatic data integrity
-- =============================================================================

-- 3.1 trg_order_status_audit — 订单状态变更记录到日志表
-- Note: Requires audit_log table. If the table does not exist, skip this trigger.
-- For simplicity, this is a placeholder. The Go code handles audit logging.

-- 3.2 trg_prevent_order_cargo_orphan — 删除订单时阻止货物残留
DELIMITER //

CREATE OR REPLACE TRIGGER trg_shipping_order_before_update_delete_time
BEFORE UPDATE ON shipping_order
FOR EACH ROW
BEGIN
    IF NEW.delete_time IS NOT NULL AND OLD.delete_time IS NULL THEN
        -- 同步软删除关联的订单货物
        UPDATE order_cargo
        SET delete_time = NEW.delete_time
        WHERE order_id = OLD.order_id AND delete_time IS NULL;
    END IF;
END //

DELIMITER ;


-- =============================================================================
-- 4. INDEX RECOMMENDATIONS (for manual review)
-- These indexes improve query performance without changing table structure.
-- Uncomment and run if needed:
-- =============================================================================

-- CREATE INDEX idx_usage_voyage_query ON segment_capacity_usage(line_id, vessel_id, voyage_date);
-- CREATE INDEX idx_order_shipper_status_time ON shipping_order(shipper_company_id, order_status, create_time);
-- CREATE INDEX idx_cargonote_voyage_query ON voyage_cargo_note(line_id, vessel_id, voyage_date, sequence_no, operation_type);

-- =============================================================================
-- End of optimizations
-- =============================================================================
