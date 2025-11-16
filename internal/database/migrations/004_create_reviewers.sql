-- +goose Up
CREATE TABLE reviewers (
    pull_request_id VARCHAR(100) REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
    user_id VARCHAR(50) REFERENCES users(user_id),
    PRIMARY KEY (pull_request_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS reviewers;