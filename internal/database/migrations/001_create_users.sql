-- +goose Up
CREATE TABLE users (
    user_id VARCHAR(50) PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    team_name VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true
);

-- +goose Down
DROP TABLE IF EXISTS users;