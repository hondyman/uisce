ALTER TABLE public.tenant_product_datasource 
    ADD CONSTRAINT tenant_product_datasource_connection_fk 
    FOREIGN KEY (connection_id) 
    REFERENCES public.connections(id) 
    ON DELETE SET NULL ON UPDATE CASCADE;
