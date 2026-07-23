-- Temporal Server initialization script
-- Creates temporal and temporal_visibility databases

-- Create main Temporal database
CREATE DATABASE IF NOT EXISTS temporal;

-- Create visibility database
CREATE DATABASE IF NOT EXISTS temporal_visibility;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE temporal TO postgres;
GRANT ALL PRIVILEGES ON DATABASE temporal_visibility TO postgres;
