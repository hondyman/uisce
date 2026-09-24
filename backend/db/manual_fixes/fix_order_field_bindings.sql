BEGIN;

-- Repoint each Order field's binding to the real /orm/order/* catalog node
-- (they were pointing at a fictional /oms/orders/* namespace with no
-- physical table, which is why only fields whose node_name happened to
-- match a real column name ever showed up in GET .../data).
UPDATE field_bindings fb
SET source_node_id = cn.id, binding_status = 'RESOLVED', updated_at = now()
FROM business_object_fields bof, catalog_node cn
WHERE fb.field_id = bof.id
  AND bof.bo_id = (SELECT id FROM business_objects WHERE bo_key = 'order')
  AND cn.qualified_path = CASE bof.technical_name
    WHEN 'avg_fill_price'   THEN '/orm/order/avg_price'
    WHEN 'created_at'       THEN '/orm/order/created_at'
    WHEN 'algo_params'      THEN '/orm/order/custom_attributes'
    WHEN 'filled_qty'       THEN '/orm/order/executed_qty'
    WHEN 'id'               THEN '/orm/order/id'
    WHEN 'leaves_qty'       THEN '/orm/order/leaves_qty'
    WHEN 'limit_price'      THEN '/orm/order/limit_price'
    WHEN 'order_type_id'    THEN '/orm/order/order_type'
    WHEN 'security_id'      THEN '/orm/order/sec_id'
    WHEN 'side'             THEN '/orm/order/side'
    WHEN 'status_id'        THEN '/orm/order/status'
    WHEN 'quantity'         THEN '/orm/order/target_qty'
    WHEN 'time_in_force_id' THEN '/orm/order/time_in_force'
    WHEN 'effective_at'     THEN '/orm/order/trade_date'
    WHEN 'updated_at'       THEN '/orm/order/updated_at'
  END;

-- Also fix the technical_name values themselves to match the real column
-- names 1:1 (several were fabricated: order_type_id -> order_type,
-- security_id -> sec_id, status_id -> status, quantity -> target_qty,
-- time_in_force_id -> time_in_force, effective_at -> trade_date,
-- avg_fill_price -> avg_price, filled_qty -> executed_qty,
-- algo_params -> custom_attributes) so the field's declared column name
-- and the resolved catalog node agree.
UPDATE business_object_fields
SET technical_name = CASE technical_name
    WHEN 'avg_fill_price'   THEN 'avg_price'
    WHEN 'order_type_id'    THEN 'order_type'
    WHEN 'security_id'      THEN 'sec_id'
    WHEN 'status_id'        THEN 'status'
    WHEN 'quantity'         THEN 'target_qty'
    WHEN 'time_in_force_id' THEN 'time_in_force'
    WHEN 'effective_at'     THEN 'trade_date'
    WHEN 'filled_qty'       THEN 'executed_qty'
    WHEN 'algo_params'      THEN 'custom_attributes'
    ELSE technical_name
  END,
  data_type = CASE technical_name
    WHEN 'order_type_id'    THEN 'text'
    WHEN 'status_id'        THEN 'text'
    WHEN 'time_in_force_id' THEN 'text'
    ELSE data_type
  END
WHERE bo_id = (SELECT id FROM business_objects WHERE bo_key = 'order');

COMMIT;
