-- name: CreateFeedFollow :one
-- Add a CreateFeedFollow query. It will be a deceptively complex SQL query. 
-- It should insert a feed follow record, but then return all the fields from 
-- the feed follow as well as the names of the linked user and feed. I'll add a tip at the bottom of this lesson if you need it.
-- name: CreateFeedFollow :one
WITH inserted AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING *
)
SELECT 
    inserted.id,
    inserted.created_at,
    inserted.updated_at,
    inserted.user_id,
    inserted.feed_id,
    users.name AS user_name,
    feeds.name AS feed_name
FROM inserted
JOIN users ON inserted.user_id = users.id
JOIN feeds ON inserted.feed_id = feeds.id;

-- name: GetFeedFollowsByUserID :many
SELECT 
    ff.id,
    ff.created_at,
    ff.updated_at,
    ff.user_id,
    ff.feed_id,
    u.name AS user_name,
    f.name AS feed_name
FROM feed_follows ff
JOIN users u ON ff.user_id = u.id
JOIN feeds f ON ff.feed_id = f.id
WHERE ff.user_id = $1
ORDER BY ff.created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteFeedFollowByID :exec
DELETE FROM feed_follows
WHERE id = $1;


-- name: DeleteAllFeedFollows :exec
DELETE FROM feed_follows;

-- name: DeleteFeedFollowByUserIDAndFeedID :exec
DELETE FROM feed_follows
WHERE feed_id = $1 AND user_id = $2;


