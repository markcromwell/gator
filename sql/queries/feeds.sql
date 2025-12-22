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
