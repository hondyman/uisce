-- Fix cascading lookups for Semantic Terms
-- Adds 'cascade_from' metadata to 'data_type' and 'format' properties

DO $$
DECLARE
    r RECORD;
    new_props jsonb;
    prop jsonb;
    i int;
BEGIN
    -- Find the semantic_term node type
    FOR r IN SELECT id, properties, config FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'
    LOOP
        -- 1. Update the main 'properties' column
        IF r.properties IS NOT NULL AND jsonb_array_length(r.properties) > 0 THEN
            new_props := '[]'::jsonb;
            FOR i IN 0..jsonb_array_length(r.properties) - 1
            LOOP
                prop := r.properties->i;
                IF prop->>'name' = 'data_type' THEN
                    prop := prop || '{"cascade_from": "semantic_type"}'::jsonb;
                ELSIF prop->>'name' = 'format' THEN
                    prop := prop || '{"cascade_from": "data_type"}'::jsonb;
                END IF;
                new_props := new_props || prop;
            END LOOP;
            
            UPDATE catalog_node_type SET properties = new_props WHERE id = r.id;
        END IF;

        -- 2. Update the 'config' column if it contains properties
        -- (Some implementations might store it in config['properties'])
        IF r.config IS NOT NULL AND r.config ? 'properties' THEN
             -- This part is more complex to update in place with simple JSONB ops if it's deep
             -- But for now, let's assume the main 'properties' column is the source of truth as per the Go code
             -- which prefers config but falls back to properties.
             -- If we updated 'properties' column, we should probably sync it to config if it exists there too.
             -- For safety, let's just update the 'properties' column which is what the Go code reads into the struct
             -- if config doesn't have it, OR if config has it, we should update that too.
             
             -- Let's just print a notice if config has properties, as handling both might be redundant 
             -- if the app logic prefers one. The Go code:
             -- "Prefer `config` when it has properties, otherwise use the properties column directly."
             -- So we MUST update config if it exists.
             
             new_props := '[]'::jsonb;
             -- Extract properties from config
             IF jsonb_typeof(r.config->'properties') = 'array' THEN
                 FOR i IN 0..jsonb_array_length(r.config->'properties') - 1
                 LOOP
                     prop := (r.config->'properties')->i;
                     IF prop->>'name' = 'data_type' THEN
                         prop := prop || '{"cascade_from": "semantic_type"}'::jsonb;
                     ELSIF prop->>'name' = 'format' THEN
                         prop := prop || '{"cascade_from": "data_type"}'::jsonb;
                     END IF;
                     new_props := new_props || prop;
                 END LOOP;
                 
                 UPDATE catalog_node_type 
                 SET config = jsonb_set(config, '{properties}', new_props)
                 WHERE id = r.id;
             END IF;
        END IF;
        
    END LOOP;
END $$;
