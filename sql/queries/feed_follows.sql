-- name: CreateFeedFollow :one
WITH new_feed_follow AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING *
)
SELECT 
    new_feed_follow.id,
    new_feed_follow.created_at,
    new_feed_follow.updated_at,
    new_feed_follow.user_id,
    new_feed_follow.feed_id,
    users.name AS user_name,
    feeds.name AS feed_name
FROM new_feed_follow
JOIN users ON new_feed_follow.user_id = users.id
JOIN feeds ON new_feed_follow.feed_id = feeds.id;

-- name: GetFeedFollowsForUser :many
WITH user_feed_follows AS (
    SELECT * FROM feed_follows WHERE feed_follows.user_id = $1
)
SELECT 
    user_feed_follows.id,
    user_feed_follows.created_at,
    user_feed_follows.updated_at,
    user_feed_follows.user_id,
    user_feed_follows.feed_id,
    users.name AS user_name,
    feeds.name AS feed_name
FROM user_feed_follows
JOIN users ON user_feed_follows.user_id = users.id
JOIN feeds ON user_feed_follows.feed_id = feeds.id;
