-- name: GetReviewStats :many
SELECT u.user_id, u.username, COUNT(r.pull_request_id) as review_count
FROM users u
LEFT JOIN reviewers r ON u.user_id = r.user_id
GROUP BY u.user_id, u.username
ORDER BY review_count DESC;

-- name: GetPRAssignmentStats :many
SELECT 
    pr.pull_request_id,
    pr.pull_request_name,
    COUNT(r.user_id) as reviewers_count
FROM pull_requests pr
LEFT JOIN reviewers r ON pr.pull_request_id = r.pull_request_id
GROUP BY pr.pull_request_id, pr.pull_request_name
ORDER BY reviewers_count DESC;

-- name: DeactivateTeamUsers :exec
UPDATE users 
SET is_active = false 
WHERE team_name = $1;

-- name: GetOpenPRsWithTeamReviewers :many
SELECT DISTINCT pr.pull_request_id
FROM pull_requests pr
JOIN reviewers r ON pr.pull_request_id = r.pull_request_id
JOIN users u ON r.user_id = u.user_id
WHERE pr.status = 'OPEN' 
AND u.team_name = $1 
AND u.is_active = true;

-- name: GetPRReviewersFromTeam :many
SELECT r.user_id
FROM reviewers r
JOIN users u ON r.user_id = u.user_id
WHERE r.pull_request_id = $1 
AND u.team_name = $2 
AND u.is_active = true;

-- name: GetActiveUsersFromTeam :many
SELECT user_id, username, team_name, is_active
FROM users 
WHERE team_name = $1 AND is_active = true;

-- name: GetAllTeams :many
SELECT team_name FROM teams;