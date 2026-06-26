-- Insert additional technical database abbreviations into sml.abbreviation_lookup
-- This script adds data type suffixes and system prefixes/suffixes commonly used in database column naming

-- Data Type & Format Suffixes
INSERT INTO sml.abbreviation_lookup (abbreviation, full_word, notes) VALUES
('AMT', 'Amount', 'Numeric field representing a monetary value'),
('CD', 'Code', 'Short alphanumeric value representing a state or category'),
('DT', 'Date', 'Field containing date in YYYY-MM-DD format'),
('DTTM', 'Datetime', 'Field containing both date and time, often with timezone'),
('FLG', 'Flag', 'Boolean-like field indicating true/false state'),
('IND', 'Indicator', 'Field indicating a specific condition or state'),
('NBR', 'Number', 'Generic numeric field'),
('PCT', 'Percent', 'Numeric field representing a percentage'),
('QTY', 'Quantity', 'Numeric field representing a count or quantity'),
('RT', 'Rate', 'Numeric field representing a rate (exchange rate, interest rate, etc.)'),
('STR', 'String', 'Text or character field'),
('TM', 'Time', 'Field containing time in HH:MM:SS format'),

-- System & Operational Prefixes/Suffixes
('EFF', 'Effective', 'Date/timestamp when a record becomes active or valid'),
('EXP', 'Expiration', 'Date/timestamp when a record is no longer valid'),
('EXT', 'External', 'Prefix indicating identifier from external system'),
('FKEY', 'Foreign Key', 'Column that is a foreign key linking to another table'),
('FK', 'Foreign Key', 'Column that is a foreign key linking to another table'),
('KEY', 'Key', 'Primary or unique key for the entity'),
('PKEY', 'Primary Key', 'Column that is the primary key for the table'),
('PK', 'Primary Key', 'Column that is the primary key for the table'),
('SRC', 'Source', 'Origin of the data (vendor, system, etc.)'),
('TYP', 'Type', 'Describes the type of entity or record')

ON CONFLICT (abbreviation)
DO UPDATE SET
    full_word = EXCLUDED.full_word,
    notes = EXCLUDED.notes;

-- Verify the insertions
SELECT COUNT(*) as total_abbreviations FROM sml.abbreviation_lookup;
SELECT abbreviation, full_word FROM sml.abbreviation_lookup WHERE abbreviation IN ('AMT', 'CD', 'DT', 'FLG', 'ID', 'KEY', 'SRC', 'TYP') ORDER BY abbreviation;