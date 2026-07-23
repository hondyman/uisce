#!/bin/bash
export PGPASSWORD=postgres
psql -h localhost -U postgres -d alpha -f backend/migrations/20260126_create_missing_relationship_tables.sql
