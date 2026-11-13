-- ==========================================
-- TAG & DECISION_TAG QUERIES
-- ==========================================

-- name: CreateTag :one
INSERT INTO tags (user_id, name, color_hex)
VALUES ($1, $2, $3)
RETURNING id, user_id, name, color_hex;

-- name: GetTagByName :one
SELECT id, user_id, name, color_hex
FROM tags
WHERE user_id = $1 AND name = $2;

-- name: GetTagByID :one
SELECT id, user_id, name, color_hex
FROM tags
WHERE id = $1 AND user_id = $2;

-- name: GetTagsByUserID :many
SELECT id, user_id, name, color_hex
FROM tags
WHERE user_id = $1
ORDER BY name ASC;

-- name: DeleteTagByID :exec
DELETE FROM tags
WHERE id = $1 AND user_id = $2;

-- name: AddTagToDecision :execrows
INSERT INTO decision_tags (decision_id, tag_id)
SELECT @decision_id, tags.id FROM tags WHERE tags.id = @tag_id AND tags.user_id = @user_id
ON CONFLICT DO NOTHING;

-- name: RemoveTagFromDecision :exec
DELETE FROM decision_tags dt
USING decisions d
WHERE dt.decision_id = d.id AND dt.decision_id = $1 AND dt.tag_id = $2 AND d.user_id = $3;

-- name: GetTagsForDecision :many
SELECT t.id, t.name, t.color_hex
FROM tags t
JOIN decision_tags dt ON dt.tag_id = t.id
WHERE dt.decision_id = $1
ORDER BY t.name ASC;
