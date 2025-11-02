-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES (
           $1,
           CURRENT_TIMESTAMP,
           CURRENT_TIMESTAMP,
           $2,
           $3,
           $4
       )
RETURNING *;


-- name: GetAllFeeds :many
SELECT *
FROM feeds;


-- name: GetFeedByUrl :one
SELECT *
FROM feeds
WHERE url = $1;


-- name: MarkFeedFetched :exec
UPDATE feeds
SET last_fetched_at = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;


-- name: GetNextFeedToFetch :one
SELECT *
FROM feeds
ORDER BY last_fetched_at NULLS FIRST
LIMIT 1;