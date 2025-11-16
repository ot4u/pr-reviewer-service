-- name: CreatePullRequest :one
INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status) 
VALUES ($1, $2, $3, 'OPEN') 
RETURNING pull_request_id, pull_request_name, author_id, status;

-- name: GetPullRequestByID :one
SELECT pull_request_id, pull_request_name, author_id, status, merged_at 
FROM pull_requests 
WHERE pull_request_id = $1;

-- name: MergePullRequest :one
UPDATE pull_requests 
SET status = 'MERGED', 
    merged_at = CASE 
        WHEN status = 'OPEN' THEN NOW()  
        ELSE merged_at                    
    END
WHERE pull_request_id = $1
RETURNING pull_request_id, pull_request_name, author_id, status, merged_at;

-- name: PRExists :one
SELECT COUNT(*) FROM pull_requests WHERE pull_request_id = $1;

-- name: IsPRMerged :one
SELECT status = 'MERGED' as is_merged 
FROM pull_requests 
WHERE pull_request_id = $1;