-- name: LockTopicOwner :one
SELECT u.id FROM public.users u WHERE u.singleton FOR NO KEY UPDATE;

-- name: FindTopicByName :one
SELECT id FROM public.topics WHERE user_id = $1 AND name_key = $2;

-- name: GetTopic :one
SELECT * FROM public.topics WHERE topics.id = $1 AND user_id = (SELECT u.id FROM public.users u WHERE u.singleton);

-- name: InsertTopic :one
INSERT INTO public.topics (id, user_id, name, name_key, light_background, light_text, dark_background, dark_text)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateTopic :one
UPDATE public.topics
SET name = sqlc.arg(name), name_key = sqlc.arg(name_key),
    light_background = sqlc.arg(light_background), light_text = sqlc.arg(light_text),
    dark_background = sqlc.arg(dark_background), dark_text = sqlc.arg(dark_text),
    updated_at = clock_timestamp()
WHERE topics.id = sqlc.arg(id) AND user_id = (SELECT u.id FROM public.users u WHERE u.singleton)
RETURNING *;

-- name: DeleteCustomTopic :exec
DELETE FROM public.topics WHERE topics.id = $1 AND user_id = (SELECT u.id FROM public.users u WHERE u.singleton)
AND system_key IS NULL;

-- name: CountTopicItems :one
SELECT count(*) FROM public.item_topics
WHERE user_id = (SELECT u.id FROM public.users u WHERE u.singleton) AND topic_id = $1;
