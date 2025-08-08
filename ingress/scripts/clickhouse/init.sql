-- ClickHouse initialization script for Docker
-- This runs when the container starts for the first time

-- Create the database
CREATE DATABASE IF NOT EXISTS trellis;

-- Create user (if not exists)
-- Note: This might need to be done manually depending on ClickHouse version
-- CREATE USER IF NOT EXISTS 'trellis' IDENTIFIED BY 'trellis_dev';
-- GRANT ALL ON trellis.* TO 'trellis';
