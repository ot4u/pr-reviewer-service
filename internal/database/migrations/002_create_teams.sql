-- +goose Up
CREATE TABLE teams (
    team_name VARCHAR(100) PRIMARY KEY
);

-- +goose Down
DROP TABLE IF EXISTS teams;