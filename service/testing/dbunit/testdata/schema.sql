CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL DEFAULT 'default', active INTEGER DEFAULT 1);
CREATE INDEX users_name ON users(name);
