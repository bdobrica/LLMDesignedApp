CREATE KEYSPACE IF NOT EXISTS user_management WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

USE user_management;

-- Create a table to store user information in the `user_management` keyspace
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    username TEXT,
    email TEXT,
    password TEXT,
    email_verified BOOLEAN,
    verification_token TEXT,
    reset_token TEXT,
    created_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS ON users(username);
CREATE INDEX IF NOT EXISTS ON users(email);
CREATE INDEX IF NOT EXISTS ON users(verification_token);

-- Create a table to store refresh tokens in the `user_management` keyspace
CREATE TABLE IF NOT EXISTS refresh_tokens (
    "token" TEXT PRIMARY KEY,
    user_id UUID,
    expires_at TIMESTAMP
);
