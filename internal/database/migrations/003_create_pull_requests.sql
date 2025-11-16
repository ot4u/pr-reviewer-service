-- +goose Up
CREATE TABLE pull_requests (
    pull_request_id VARCHAR(100) PRIMARY KEY,
    pull_request_name VARCHAR(200) NOT NULL,
    author_id VARCHAR(50) NOT NULL REFERENCES users(user_id),
    status VARCHAR(20) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'MERGED')),
    merged_at TIMESTAMP WITH TIME ZONE NULL
);

-- +goose Down
DROP TABLE IF EXISTS pull_requests;