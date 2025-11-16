-- name: AssignReviewer :exec
INSERT INTO reviewers (pull_request_id, user_id) 
VALUES ($1, $2);

-- name: GetPRReviewers :many
SELECT user_id FROM reviewers 
WHERE pull_request_id = $1;

-- name: RemoveReviewer :exec
DELETE FROM reviewers 
WHERE pull_request_id = $1 AND user_id = $2;

-- name: IsUserReviewer :one
SELECT COUNT(*) FROM reviewers 
WHERE pull_request_id = $1 AND user_id = $2;

-- name: GetUserAssignedPRs :many
SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
FROM pull_requests pr
JOIN reviewers r ON pr.pull_request_id = r.pull_request_id
WHERE r.user_id = $1;

-- name: CountReviewers :one
SELECT COUNT(*) FROM reviewers WHERE pull_request_id = $1;