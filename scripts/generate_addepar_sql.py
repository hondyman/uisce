import json
import uuid

def generate_sql():
    with open('backend/internal/metadata/addepar_models.json', 'r') as f:
        models = json.load(f)

    sql_statements = []
    
    # Header
    sql_statements.append("-- Seed Addepar Business Object Definitions")
    sql_statements.append("BEGIN;")

    for model in models:
        model_type = model.get("model_type")
        display_name = model.get("display_name")
        desc = model.get("description", "").replace("'", "''")
        
        # Construct config JSON
        config = {
            "display_name": display_name,
            "ownership_type": model.get("ownership_type"),
            "suggested_attributes": model.get("suggested_attributes", [])
        }
        config_json = json.dumps(config).replace("'", "''")

        # UUID generation (deterministic based on model_type for idempotency if needed, 
        # but here we rely on the UNIQUE constraint on (tenant_id, catalog_type_name))
        
        # Upsert statement
        stmt = f"""
        INSERT INTO catalog_node_types (tenant_id, catalog_type_name, description, config)
        VALUES (
            '00000000-0000-0000-0000-000000000000', -- Default Tenant
            '{model_type}',
            '{desc}',
            '{config_json}'
        )
        ON CONFLICT (tenant_id, catalog_type_name) 
        DO UPDATE SET 
            description = EXCLUDED.description,
            config = EXCLUDED.config,
            updated_at = CURRENT_TIMESTAMP;
        """
        sql_statements.append(stmt)

    sql_statements.append("COMMIT;")
    
    return "\n".join(sql_statements)

if __name__ == "__main__":
    print(generate_sql())
