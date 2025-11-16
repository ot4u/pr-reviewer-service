-- +goose Up
-- Индекс для поиска активных пользователей в команде (автоназначение ревьюверов)
CREATE INDEX IF NOT EXISTS idx_users_team_active 
ON users(team_name, is_active) 
WHERE is_active = true;

-- Индекс для поиска PR по автору (создание PR, проверка автора)
CREATE INDEX IF NOT EXISTS idx_pr_author 
ON pull_requests(author_id);

-- Индекс для поиска PR по статусу (merge операция, проверка статуса)
CREATE INDEX IF NOT EXISTS idx_pr_status 
ON pull_requests(status);

-- Индекс для поиска PR назначенных пользователю (/users/getReview эндпоинт)
CREATE INDEX IF NOT EXISTS idx_reviewers_user_id 
ON reviewers(user_id);

-- Индекс для проверки уникальности PR (создание PR)
CREATE UNIQUE INDEX IF NOT EXISTS idx_pr_id 
ON pull_requests(pull_request_id);

-- +goose Down
DROP INDEX IF EXISTS idx_users_team_active;
DROP INDEX IF EXISTS idx_pr_author;
DROP INDEX IF EXISTS idx_pr_status;
DROP INDEX IF EXISTS idx_reviewers_user_id;
DROP INDEX IF EXISTS idx_pr_id;