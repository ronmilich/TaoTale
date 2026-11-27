-- name: AddItem :one
INSERT INTO public.library_items (
    user_id, language_code, title, source_text, description, notes, difficulty, is_favorite, voice_id, speaking_rate
)
SELECT u.id, l.code, sqlc.narg(title), sqlc.arg(source_text), sqlc.narg(description), sqlc.narg(notes),
       sqlc.narg(difficulty), sqlc.arg(is_favorite), sqlc.narg(voice_id), sqlc.narg(speaking_rate)
FROM public.users u CROSS JOIN public.learning_languages l
WHERE u.singleton AND l.enabled AND l.code = sqlc.arg(language_code)
RETURNING *;

-- name: GetItem :one
SELECT * FROM public.library_items
WHERE library_items.id = $1 AND user_id = (SELECT u.id FROM public.users u WHERE u.singleton);

-- name: LockItem :one
SELECT * FROM public.library_items
WHERE library_items.id = $1 AND user_id = (SELECT u.id FROM public.users u WHERE u.singleton)
FOR UPDATE;

-- name: UpdateItem :one
UPDATE public.library_items
SET title = sqlc.narg(title), source_text = sqlc.arg(source_text), description = sqlc.narg(description),
    notes = sqlc.narg(notes), difficulty = sqlc.narg(difficulty),
    voice_id = sqlc.narg(voice_id), speaking_rate = sqlc.narg(speaking_rate),
    version = version + 1, updated_at = clock_timestamp()
WHERE library_items.id = sqlc.arg(id) AND user_id = (SELECT u.id FROM public.users u WHERE u.singleton)
  AND version = sqlc.arg(expected_version)
RETURNING *;

-- name: SetItemFavorite :one
UPDATE public.library_items
SET is_favorite = sqlc.arg(is_favorite)
WHERE library_items.id = sqlc.arg(id) AND user_id = (SELECT u.id FROM public.users u WHERE u.singleton)
RETURNING id, is_favorite;

-- name: DeleteItem :execrows
DELETE FROM public.library_items
WHERE library_items.id = $1 AND user_id = (SELECT u.id FROM public.users u WHERE u.singleton);

-- name: PageItems :many
SELECT * FROM public.library_items
WHERE user_id = (SELECT u.id FROM public.users u WHERE u.singleton) AND language_code = sqlc.arg(language_code)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_size)::int OFFSET sqlc.arg(page_offset)::bigint;

-- name: CountItems :one
SELECT count(*) FROM public.library_items
WHERE user_id = (SELECT u.id FROM public.users u WHERE u.singleton) AND language_code = $1;

-- name: GetItemTopics :many
SELECT item_id, topic_id FROM public.item_topics
WHERE user_id = (SELECT u.id FROM public.users u WHERE u.singleton) AND item_id = ANY(sqlc.arg(item_ids)::uuid[])
ORDER BY item_id, topic_id;

-- name: ClearItemTopics :exec
DELETE FROM public.item_topics WHERE item_id = $1 AND user_id = (SELECT u.id FROM public.users u WHERE u.singleton);

-- name: AddItemTopics :execrows
INSERT INTO public.item_topics (user_id, item_id, topic_id)
SELECT user_id, sqlc.arg(item_id), id FROM public.topics
WHERE user_id = (SELECT u.id FROM public.users u WHERE u.singleton)
  AND id = ANY(sqlc.arg(topic_ids)::uuid[]);

-- name: LockOwnedTopics :many
SELECT topics.id FROM public.topics
WHERE user_id = (SELECT u.id FROM public.users u WHERE u.singleton)
  AND topics.id = ANY(sqlc.arg(topic_ids)::uuid[])
ORDER BY topics.id
FOR KEY SHARE;
