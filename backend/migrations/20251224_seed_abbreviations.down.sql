-- Rollback: Remove seeded abbreviations
DELETE FROM abbreviations 
WHERE abbreviation IN (
    'CEO', 'CFO', 'COO', 'CTO', 'HR', 'IT', 'R&D', 'KPI', 'ROI', 'SLA', 
    'API', 'SQL', 'JSON', 'REST', 'CRUD'
);
