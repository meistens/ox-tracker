-- Create a new user and database for your app
-- CREATE USER oxbreeze
-- WITH
--     PASSWORD 'passwrd';

CREATE DATABASE oxtracker OWNER oxbreeze;

-- (Optional) Connect to the new database and grant privileges
\connect oxtracker

-- Grant all privileges on the public schema to the app user (default for new DBs)
GRANT ALL PRIVILEGES ON SCHEMA public TO oxbreeze;

-- Grant all privileges on all tables, sequences, and functions in the public schema
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO oxbreeze;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO oxbreeze;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO oxbreeze;

-- Ensure future tables, sequences, and functions are accessible
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO oxbreeze;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON SEQUENCES TO oxbreeze;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON FUNCTIONS TO oxbreeze;

-- You can add more users/databases as needed
