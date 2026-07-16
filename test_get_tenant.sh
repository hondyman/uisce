#!/bin/bash
curl -s "http://127.0.0.1:3000/api/tenants/99e99e99-99e9-49e9-89e9-99e99e99e999" | jq '.tenant.tenant_instances[0].products[0].datasources[0] | keys'
