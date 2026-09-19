-- Populate display labels on Order-chain BO fields so forms and tables
-- show "Average Price" instead of AveragePrice / avg_price.
-- Only fills empty or technical-name-equal labels; tenant overlays keep custom names.

UPDATE public.business_object_fields f
SET display_name = m.label
FROM (VALUES
  ('status', 'Status'),
  ('side', 'Side'),
  ('order_type', 'Order Type'),
  ('ordertype', 'Order Type'),
  ('time_in_force', 'Time In Force'),
  ('timeinforce', 'Time In Force'),
  ('tif', 'Time In Force'),
  ('trade_date', 'Trade Date'),
  ('tradedate', 'Trade Date'),
  ('target_qty', 'Target Quantity'),
  ('targetqty', 'Target Quantity'),
  ('targetquantity', 'Target Quantity'),
  ('executed_qty', 'Executed Quantity'),
  ('executedqty', 'Executed Quantity'),
  ('executedquantity', 'Executed Quantity'),
  ('leaves_qty', 'Leaves Quantity'),
  ('leavesqty', 'Leaves Quantity'),
  ('leavesquantity', 'Leaves Quantity'),
  ('limit_price', 'Limit Price'),
  ('limitprice', 'Limit Price'),
  ('avg_price', 'Average Price'),
  ('avgprice', 'Average Price'),
  ('averageprice', 'Average Price'),
  ('sec_id', 'Security'),
  ('securityid', 'Security'),
  ('securitiesid', 'Security'),
  ('broker_id', 'Broker'),
  ('brokerid', 'Broker'),
  ('account_id', 'Account'),
  ('accountid', 'Account'),
  ('fix_clordid', 'FIX ClOrdID'),
  ('fixclordid', 'FIX ClOrdID'),
  ('routed_qty', 'Routed Quantity'),
  ('routedqty', 'Routed Quantity'),
  ('exec_qty', 'Exec Quantity'),
  ('execqty', 'Exec Quantity'),
  ('exec_price', 'Exec Price'),
  ('execprice', 'Exec Price'),
  ('exec_time', 'Exec Time'),
  ('exectime', 'Exec Time'),
  ('allocated_qty', 'Allocated Quantity'),
  ('allocatedqty', 'Allocated Quantity'),
  ('venue_id', 'Venue'),
  ('venueid', 'Venue'),
  ('created_at', 'Created'),
  ('createdat', 'Created'),
  ('updated_at', 'Updated'),
  ('updatedat', 'Updated')
) AS m(key, label)
WHERE lower(replace(COALESCE(f.technical_name, f.field_name), '_', '')) = replace(m.key, '_', '')
  AND (f.display_name IS NULL OR btrim(f.display_name) = '' OR f.display_name = f.field_name);
