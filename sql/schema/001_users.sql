-- +goose Up
-- Enable the extension to generate UUIDs automatically
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    name TEXT NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE users;
DROP EXTENSION IF EXISTS "pgcrypto";
