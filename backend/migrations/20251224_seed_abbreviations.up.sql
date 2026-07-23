-- Seed abbreviations table with common business abbreviations
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'abbreviations' AND column_name = 'expansion') THEN
    INSERT INTO abbreviations (abbreviation, expansion, context) VALUES
      ('CEO', 'Chief Executive Officer', 'Executive'),
      ('CFO', 'Chief Financial Officer', 'Executive'),
      ('COO', 'Chief Operating Officer', 'Executive'),
      ('CTO', 'Chief Technology Officer', 'Executive'),
      ('HR', 'Human Resources', 'Department'),
      ('IT', 'Information Technology', 'Department'),
      ('RD', 'Research and Development', 'Department'),
      ('KPI', 'Key Performance Indicator', 'Business'),
      ('ROI', 'Return on Investment', 'Business'),
      ('SLA', 'Service Level Agreement', 'Business'),
      ('API', 'Application Programming Interface', 'Technology'),
      ('SQL', 'Structured Query Language', 'Technology'),
      ('JSON', 'JavaScript Object Notation', 'Technology'),
      ('REST', 'Representational State Transfer', 'Technology'),
      ('CRUD', 'Create Read Update Delete', 'Technology'),
      -- Geographic/Location
      ('CNTRY', 'Country', 'Geographic'),
      ('CTY', 'City', 'Geographic'),
      ('ST', 'State', 'Geographic'),
      ('ADDR', 'Address', 'Geographic'),
      ('ZIP', 'Zip Code', 'Geographic'),
      ('REGN', 'Region', 'Geographic'),
      -- Financial
      ('AMT', 'Amount', 'Financial'),
      ('BAL', 'Balance', 'Financial'),
      ('CURR', 'Currency', 'Financial'),
      ('ACCT', 'Account', 'Financial'),
      ('TXN', 'Transaction', 'Financial'),
      ('PMT', 'Payment', 'Financial'),
      ('FX', 'Foreign Exchange', 'Financial'),
      -- Common abbreviations
      ('CD', 'Code', 'Common'),
      ('ID', 'Identifier', 'Common'),
      ('DT', 'Date', 'Common'),
      ('NUM', 'Number', 'Common'),
      ('DESC', 'Description', 'Common'),
      ('MAX', 'Maximum', 'Common'),
      ('MIN', 'Minimum', 'Common'),
      ('AVG', 'Average', 'Common'),
      ('TOT', 'Total', 'Common'),
      ('VAL', 'Value', 'Common'),
      ('QTY', 'Quantity', 'Common'),
      ('STR', 'String', 'Common'),
      ('BOOL', 'Boolean', 'Common'),
      ('PK', 'Primary Key', 'Database'),
      ('FK', 'Foreign Key', 'Database'),
      ('IDX', 'Index', 'Database'),
      ('TEMP', 'Temporary', 'Common'),
      ('OBJ', 'Object', 'Common'),
      ('SRC', 'Source', 'Common'),
      ('TGT', 'Target', 'Common')
    ON CONFLICT (abbreviation) DO NOTHING;
  END IF;
END$$;
