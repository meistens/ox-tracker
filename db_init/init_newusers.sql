-- Create a new user and database for your app
CREATE USER myappuser
WITH
    PASSWORD 'myapppassword';

CREATE DATABASE myappdb OWNER myappuser;

-- (Optional) Connect to the new database and grant privileges
\connect myappdb

-- Grant all privileges on the public schema to the app user (default for new DBs)
GRANT ALL PRIVILEGES ON SCHEMA public TO myappuser;

-- Grant all privileges on all tables, sequences, and functions in the public schema
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO myappuser;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO myappuser;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO myappuser;

-- Ensure future tables, sequences, and functions are accessible
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON TABLES TO myappuser;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON SEQUENCES TO myappuser;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL PRIVILEGES ON FUNCTIONS TO myappuser;

-- You can add more users/databases as needed
