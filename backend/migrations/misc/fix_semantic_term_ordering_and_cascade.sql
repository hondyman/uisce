-- Fix Semantic Term property ordering and cascade references
-- 1. Moves 'semantic_type' to top (Order 1)
-- 2. Ensures 'datatype' is Order 2 and cascades from 'semantic_type'
-- 3. Ensures 'format' is Order 3 and cascades from 'datatype' (fixing the previous 'data_type' error)
-- 4. Moves conflicting properties ('sql', 'title') to lower positions

DO $$
DECLARE
    r RECORD;
    new_props jsonb;
    prop jsonb;
    i int;
BEGIN
    FOR r IN SELECT id, properties, config FROM catalog_node_type WHERE catalog_type_name = 'semantic_term'
    LOOP
        -- 1. Update properties column
        IF r.properties IS NOT NULL AND jsonb_array_length(r.properties) > 0 THEN
            new_props := '[]'::jsonb;
            FOR i IN 0..jsonb_array_length(r.properties) - 1
            LOOP
                prop := r.properties->i;
                
                IF prop->>'name' = 'semantic_type' THEN
                    prop := prop || '{"order": 1}'::jsonb;
                ELSIF prop->>'name' = 'datatype' THEN
                    prop := prop || '{"order": 2, "cascade_from": "semantic_type"}'::jsonb;
                ELSIF prop->>'name' = 'format' THEN
                    -- Fix the cascade_from to point to 'datatype' (the actual property name), not 'data_type'
                    prop := prop || '{"order": 3, "cascade_from": "datatype"}'::jsonb;
                ELSIF prop->>'name' = 'sql' THEN
                    prop := prop || '{"order": 10}'::jsonb;
                ELSIF prop->>'name' = 'title' THEN
                    prop := prop || '{"order": 11}'::jsonb;
                END IF;
                
                new_props := new_props || prop;
            END LOOP;
            
            UPDATE catalog_node_type SET properties = new_props WHERE id = r.id;
        END IF;

        -- 2. Update config column if it exists
        IF r.config IS NOT NULL AND r.config ? 'properties' THEN
             IF jsonb_typeof(r.config->'properties') = 'array' THEN
                 new_props := '[]'::jsonb;
                 FOR i IN 0..jsonb_array_length(r.config->'properties') - 1
                 LOOP
                     prop := (r.config->'properties')->i;
                     
                     IF prop->>'name' = 'semantic_type' THEN
                        prop := prop || '{"order": 1}'::jsonb;
                     ELSIF prop->>'name' = 'datatype' THEN
                        prop := prop || '{"order": 2, "cascade_from": "semantic_type"}'::jsonb;
                     ELSIF prop->>'name' = 'format' THEN
                        prop := prop || '{"order": 3, "cascade_from": "datatype"}'::jsonb;
                     ELSIF prop->>'name' = 'sql' THEN
                        prop := prop || '{"order": 10}'::jsonb;
                     ELSIF prop->>'name' = 'title' THEN
                        prop := prop || '{"order": 11}'::jsonb;
                     END IF;
                     
                     new_props := new_props || prop;
                 END LOOP;
                 
                 UPDATE catalog_node_type 
                 SET config = jsonb_set(r.config, '{properties}', new_props)
                 WHERE id = r.id;
             END IF;
        END IF;
        
    END LOOP;
END $$;
