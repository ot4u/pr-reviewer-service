-- name: CreateTeam :one
INSERT INTO teams (team_name) 
VALUES ($1) 
RETURNING team_name;

-- name: TeamExists :one
SELECT COUNT(*) FROM teams WHERE team_name = $1;