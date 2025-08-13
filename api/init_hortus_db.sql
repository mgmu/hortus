-- Creates schema and sets the search path
DROP SCHEMA IF EXISTS hortus_schema CASCADE;
CREATE SCHEMA hortus_schema;
SET search_path TO hortus_schema;

-- Creates tables
CREATE TABLE hortus_schema.plant (
       id SERIAL PRIMARY KEY,
       common_name VARCHAR(255) NOT NULL,
       generic_name VARCHAR(255),
       specific_name VARCHAR(255),
       CHECK (common_name <> '')
);

CREATE TABLE hortus_schema.plant_log (
       id SERIAL PRIMARY KEY,
       plant_id INTEGER NOT NULL,
       -- log_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
       description VARCHAR(255) NOT NULL,
       event_type INTEGER NOT NULL,
       FOREIGN KEY (plant_id) REFERENCES plant(id)
);
