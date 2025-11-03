-- name: CreatePost :one
INSERT INTO posts (id, created_at, updated_at, title, url, description, published_at, feed_id)
VALUES (
           $1,
           CURRENT_TIMESTAMP,
           CURRENT_TIMESTAMP,
           $2,
           $3,
           $4,
           $5,
           $6
       )
RETURNING *;


-- name: GetPostsForUser :many
SELECT *
FROM posts
JOIN feed_follows
ON posts.feed_id = feed_follows.feed_id
    AND feed_follows.user_id = $1
ORDER BY published_at DESC
LIMIT $2;