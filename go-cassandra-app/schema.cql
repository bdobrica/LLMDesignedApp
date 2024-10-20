CREATE KEYSPACE IF NOT EXISTS user_management WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

USE user_management;

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
