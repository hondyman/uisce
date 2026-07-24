-- 1. Create the main View Definition table
CREATE TABLE IF NOT EXISTS view_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE, -- e.g., 'order_approval_form', 'customer_details_view'
    title VARCHAR(255), -- e.g., 'Order Approval Required'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Define the components (fields) for that view
CREATE TABLE IF NOT EXISTS view_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    view_definition_id UUID NOT NULL REFERENCES view_definitions(id) ON DELETE CASCADE,
    -- The key used to map data from the workflow context to this component.
    data_key VARCHAR(255) NOT NULL,
    
    -- UI rendering hints
    component_type VARCHAR(50) NOT NULL, -- e.g., 'TextField', 'Table', 'Select', 'Button'
    label VARCHAR(255),
    "order" INT NOT NULL, -- Display order
    
    -- Optional properties for configuration, stored as flexible JSONB
    properties JSONB, -- e.g., { "readOnly": true }, { "options": [...] }

    UNIQUE(view_definition_id, data_key)
);

-- 3. Seed Metadata for "High-Value Order Approval" Form (Northwind Blueprint)
INSERT INTO view_definitions (name, title) 
VALUES ('high_value_order_approval_form', 'High-Value Order Approval')
ON CONFLICT (name) DO NOTHING;

-- Populate components (Using a CTE to get the ID safely)
WITH view_def AS (
    SELECT id FROM view_definitions WHERE name = 'high_value_order_approval_form'
)
INSERT INTO view_components (view_definition_id, data_key, component_type, label, "order", properties)
VALUES
    ((SELECT id FROM view_def), 'customer_name', 'ReadOnlyText', 'Customer', 1, '{"style": "heading"}'),
    ((SELECT id FROM view_def), 'order_date', 'ReadOnlyText', 'Order Date', 2, NULL),
    ((SELECT id FROM view_def), 'order_total', 'ReadOnlyText', 'Order Total', 3, '{"format": "currency"}'),
    ((SELECT id FROM view_def), 'requires_at', 'ReadOnlyText', 'Required At', 4, NULL),
    ((SELECT id FROM view_def), 'products_list', 'Table', 'Products', 5, 
        '{ "columns": [{"header": "Product", "key": "productName"}, {"header": "Quantity", "key": "quantity"}, {"header": "Unit Price", "key": "unitPrice"}] }'),
    ((SELECT id FROM view_def), 'approval_decision', 'Select', 'Decision', 6, 
        '{ "options": [{"label": "Approve", "value": "approved"}, {"label": "Reject", "value": "rejected"}] }'),
    ((SELECT id FROM view_def), 'approval_notes', 'TextArea', 'Notes', 7, '{ "placeholder": "Optional notes..." }'),
    ((SELECT id FROM view_def), 'submit_button', 'Button', 'Submit Decision', 8, '{ "isPrimary": true }')
ON CONFLICT (view_definition_id, data_key) DO NOTHING;
