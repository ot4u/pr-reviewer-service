-- name: GetUserByID :one
SELECT user_id, username, team_name, is_active 
FROM users 
WHERE user_id = $1;

-- name: GetActiveUsersByTeam :many
SELECT user_id, username, team_name, is_active 
FROM users 
WHERE team_name = $1 AND is_active = true 
AND user_id != $2  
ORDER BY RANDOM()
LIMIT 2;           

-- name: UpdateUserActiveStatus :one
UPDATE users 
SET is_active = $2 
WHERE user_id = $1 
RETURNING user_id, username, team_name, is_active;

-- name: GetUserTeam :one
SELECT team_name FROM users WHERE user_id = $1;

-- name: UpsertUser :one
INSERT INTO users (user_id, username, team_name, is_active)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) 
DO UPDATE SET 
    team_name = EXCLUDED.team_name 
RETURNING user_id, username, team_name, is_active;

-- name: GetAllUsersByTeam :many
SELECT user_id, username, team_name, is_active 
FROM users 
WHERE team_name = $1;