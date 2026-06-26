-- Sample Insurance Data for Testing Calculations
-- This file contains sample data structures for insurance calculations

-- Insurance Policy Data
CREATE TABLE IF NOT EXISTS insurance_policies (
    policy_id SERIAL PRIMARY KEY,
    policy_number VARCHAR(50) UNIQUE,
    customer_id INTEGER,
    product_type VARCHAR(50),
    coverage_amount DECIMAL(15,2),
    premium_amount DECIMAL(12,2),
    effective_date DATE,
    expiry_date DATE,
    status VARCHAR(20)
);

-- Claims Data
CREATE TABLE IF NOT EXISTS insurance_claims (
    claim_id SERIAL PRIMARY KEY,
    policy_id INTEGER REFERENCES insurance_policies(policy_id),
    claim_number VARCHAR(50) UNIQUE,
    claim_date DATE,
    incident_date DATE,
    claim_amount DECIMAL(12,2),
    paid_amount DECIMAL(12,2),
    reserves_held DECIMAL(12,2),
    claim_status VARCHAR(20),
    loss_type VARCHAR(50)
);

-- Expenses Data
CREATE TABLE IF NOT EXISTS insurance_expenses (
    expense_id SERIAL PRIMARY KEY,
    expense_date DATE,
    expense_type VARCHAR(50),
    amount DECIMAL(12,2),
    category VARCHAR(50)
);

-- Investment Data
CREATE TABLE IF NOT EXISTS insurance_investments (
    investment_id SERIAL PRIMARY KEY,
    investment_date DATE,
    asset_type VARCHAR(50),
    invested_amount DECIMAL(15,2),
    current_value DECIMAL(15,2),
    income_received DECIMAL(12,2)
);

-- Reserves Data
CREATE TABLE IF NOT EXISTS insurance_reserves (
    reserve_id SERIAL PRIMARY KEY,
    policy_id INTEGER REFERENCES insurance_policies(policy_id),
    reserve_date DATE,
    reserve_type VARCHAR(50),
    amount DECIMAL(12,2),
    development_year INTEGER
);

-- Sample Data Inserts
INSERT INTO insurance_policies (policy_number, customer_id, product_type, coverage_amount, premium_amount, effective_date, expiry_date, status) VALUES
('POL001', 1, 'Auto', 500000.00, 1200.00, '2024-01-01', '2025-01-01', 'Active'),
('POL002', 2, 'Home', 750000.00, 1800.00, '2024-01-01', '2025-01-01', 'Active'),
('POL003', 3, 'Life', 1000000.00, 2500.00, '2024-01-01', '2025-01-01', 'Active'),
('POL004', 4, 'Auto', 400000.00, 950.00, '2024-01-01', '2025-01-01', 'Active'),
('POL005', 5, 'Home', 600000.00, 1400.00, '2024-01-01', '2025-01-01', 'Active');

INSERT INTO insurance_claims (policy_id, claim_number, claim_date, incident_date, claim_amount, paid_amount, reserves_held, claim_status, loss_type) VALUES
(1, 'CLM001', '2024-03-15', '2024-03-10', 15000.00, 15000.00, 0.00, 'Closed', 'Collision'),
(2, 'CLM002', '2024-04-20', '2024-04-15', 25000.00, 20000.00, 5000.00, 'Open', 'Water Damage'),
(1, 'CLM003', '2024-06-10', '2024-06-05', 8000.00, 8000.00, 0.00, 'Closed', 'Theft'),
(4, 'CLM004', '2024-07-25', '2024-07-20', 12000.00, 10000.00, 2000.00, 'Open', 'Collision');

INSERT INTO insurance_expenses (expense_date, expense_type, amount, category) VALUES
('2024-01-31', 'Commission', 2400.00, 'Acquisition'),
('2024-01-31', 'Administrative', 1800.00, 'Operations'),
('2024-01-31', 'Claims Processing', 1200.00, 'Claims'),
('2024-02-29', 'Commission', 2150.00, 'Acquisition'),
('2024-02-29', 'Administrative', 1750.00, 'Operations'),
('2024-02-29', 'Claims Processing', 1350.00, 'Claims');

INSERT INTO insurance_investments (investment_date, asset_type, invested_amount, current_value, income_received) VALUES
('2024-01-01', 'Bonds', 5000000.00, 5100000.00, 75000.00),
('2024-01-01', 'Stocks', 3000000.00, 3150000.00, 45000.00),
('2024-01-01', 'Real Estate', 2000000.00, 2050000.00, 30000.00);

INSERT INTO insurance_reserves (policy_id, reserve_date, reserve_type, amount, development_year) VALUES
(1, '2024-01-01', 'Loss Reserve', 50000.00, 2024),
(2, '2024-01-01', 'Loss Reserve', 75000.00, 2024),
(3, '2024-01-01', 'Loss Reserve', 100000.00, 2024),
(1, '2024-06-30', 'Loss Reserve', 45000.00, 2024),
(2, '2024-06-30', 'Loss Reserve', 70000.00, 2024);
