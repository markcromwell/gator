-- name: CreateFeeds :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetFeed :many
SELECT *
FROM feeds
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetFeedByID :one
SELECT *
FROM feeds
WHERE id = $1;  

-- name: GetFeedByURL :one
SELECT *
FROM feeds
where url = $1;

-- name: DeleteAllFeeds :exec
DELETE FROM feeds;

-- name: DeleteFeedByUserIDAndFeedID :exec
DELETE FROM feeds
WHERE id = $1 AND user_id = $2;

-- name: MarkFeedFetched :exec
-- set last_fetched_at and updated_at to current timestamp
UPDATE feeds
SET last_fetched_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: GetNextFeedToFetch :one
SELECT *
FROM feeds
WHERE last_fetched_at IS NULL
ORDER BY last_fetched_at ASC NULLS FIRST
LIMIT 1;
