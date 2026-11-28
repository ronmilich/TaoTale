-- name: InsertVocabulary :one
INSERT INTO public.vocabulary_entries (user_id, language_code, word, normalized_word_key,
    reading, reading_system, meaning, note, status)
SELECT u.id, l.code, sqlc.arg(word), sqlc.arg(normalized_word_key),
    sqlc.narg(reading), sqlc.narg(reading_system), sqlc.narg(meaning), sqlc.narg(note), sqlc.arg(status)
FROM public.users u CROSS JOIN public.learning_languages l
WHERE u.singleton AND l.code = sqlc.arg(language_code) AND l.enabled
RETURNING *;

-- name: GetVocabulary :one
SELECT * FROM public.vocabulary_entries
WHERE vocabulary_entries.id = $1 AND user_id = (SELECT id FROM public.users WHERE singleton);

-- name: FindVocabulary :one
SELECT * FROM public.vocabulary_entries
WHERE user_id = (SELECT id FROM public.users WHERE singleton)
    AND language_code = $1 AND normalized_word_key = $2;

-- name: LockVocabularyLanguage :one
SELECT language_code FROM public.vocabulary_entries
WHERE vocabulary_entries.id = $1 AND user_id = (SELECT id FROM public.users WHERE singleton)
FOR KEY SHARE;

-- name: UpdateVocabulary :one
UPDATE public.vocabulary_entries
SET word = coalesce(sqlc.narg(word), word), normalized_word_key = coalesce(sqlc.narg(normalized_word_key), normalized_word_key),
    reading = CASE WHEN sqlc.arg(set_reading)::boolean THEN sqlc.narg(reading) ELSE reading END,
    reading_system = CASE WHEN sqlc.arg(set_reading_system)::boolean THEN sqlc.narg(reading_system) ELSE reading_system END,
    meaning = CASE WHEN sqlc.arg(set_meaning)::boolean THEN sqlc.narg(meaning) ELSE meaning END,
    note = CASE WHEN sqlc.arg(set_note)::boolean THEN sqlc.narg(note) ELSE note END,
    status = coalesce(sqlc.narg(status), status),
    updated_at = clock_timestamp()
WHERE vocabulary_entries.id = sqlc.arg(id) AND user_id = (SELECT id FROM public.users WHERE singleton)
RETURNING *;

-- name: DeleteVocabulary :execrows
DELETE FROM public.vocabulary_entries
WHERE vocabulary_entries.id = $1 AND user_id = (SELECT id FROM public.users WHERE singleton);

-- name: GetVocabularyTopics :many
SELECT vocabulary_entry_id, topic_id FROM public.vocabulary_topics
WHERE user_id = (SELECT id FROM public.users WHERE singleton)
    AND vocabulary_entry_id = ANY(sqlc.arg(ids)::uuid[])
ORDER BY vocabulary_entry_id, topic_id;

-- name: ClearVocabularyTopics :exec
DELETE FROM public.vocabulary_topics
WHERE vocabulary_entry_id = $1 AND user_id = (SELECT id FROM public.users WHERE singleton);

-- name: AddVocabularyTopics :exec
INSERT INTO public.vocabulary_topics (user_id, vocabulary_entry_id, topic_id)
SELECT v.user_id, v.id, unnest(sqlc.arg(topic_ids)::uuid[])
FROM public.vocabulary_entries v
WHERE v.id = sqlc.arg(id) AND v.user_id = (SELECT id FROM public.users WHERE singleton);

-- name: LockVocabularySource :one
SELECT language_code FROM public.library_items
WHERE library_items.id = $1 AND user_id = (SELECT id FROM public.users WHERE singleton)
FOR KEY SHARE;

-- name: LinkVocabulary :one
INSERT INTO public.vocabulary_references (user_id, language_code, vocabulary_entry_id, library_item_id, manual_linked)
SELECT v.user_id, v.language_code, v.id, sqlc.arg(item_id)::uuid, true
FROM public.vocabulary_entries v
WHERE v.id = sqlc.arg(id) AND v.user_id = (SELECT id FROM public.users WHERE singleton)
ON CONFLICT (vocabulary_entry_id, library_item_id)
DO UPDATE SET manual_linked = true
RETURNING *;

-- name: UnlinkVocabulary :exec
DELETE FROM public.vocabulary_references
WHERE vocabulary_entry_id = sqlc.arg(id) AND library_item_id = sqlc.arg(item_id)
    AND user_id = (SELECT id FROM public.users WHERE singleton);

-- name: GetVocabularyReferences :many
SELECT r.library_item_id, r.added_at, r.manual_linked, r.auto_linked, i.title
FROM public.vocabulary_references r
JOIN public.library_items i ON i.id = r.library_item_id
WHERE r.vocabulary_entry_id = $1 AND r.user_id = (SELECT id FROM public.users WHERE singleton)
ORDER BY r.added_at DESC, r.library_item_id DESC;

-- name: ItemVocabulary :many
SELECT sqlc.embed(v), r.added_at, r.manual_linked, r.auto_linked
FROM public.vocabulary_references r
JOIN public.vocabulary_entries v ON v.id = r.vocabulary_entry_id
WHERE r.library_item_id = $1 AND r.user_id = (SELECT id FROM public.users WHERE singleton)
ORDER BY r.added_at DESC, v.id DESC;

-- name: CountVocabulary :one
SELECT count(*) FROM public.vocabulary_entries v
WHERE v.user_id = (SELECT id FROM public.users WHERE singleton)
    AND v.language_code = sqlc.arg(language_code)
    AND (sqlc.arg(status)::text = '' OR v.status = sqlc.arg(status))
    AND (sqlc.arg(pattern)::text = '' OR v.word ILIKE sqlc.arg(pattern)
        OR v.reading ILIKE sqlc.arg(pattern) OR v.meaning ILIKE sqlc.arg(pattern))
    AND (coalesce(cardinality(sqlc.arg(topic_ids)::uuid[]), 0) = 0 OR EXISTS (
        SELECT 1 FROM public.vocabulary_topics vt
        WHERE vt.vocabulary_entry_id = v.id AND vt.topic_id = ANY(sqlc.arg(topic_ids))
    ));

-- name: PageVocabulary :many
SELECT v.* FROM public.vocabulary_entries v
WHERE v.user_id = (SELECT id FROM public.users WHERE singleton)
    AND v.language_code = sqlc.arg(language_code)
    AND (sqlc.arg(status)::text = '' OR v.status = sqlc.arg(status))
    AND (sqlc.arg(pattern)::text = '' OR v.word ILIKE sqlc.arg(pattern)
        OR v.reading ILIKE sqlc.arg(pattern) OR v.meaning ILIKE sqlc.arg(pattern))
    AND (coalesce(cardinality(sqlc.arg(topic_ids)::uuid[]), 0) = 0 OR EXISTS (
        SELECT 1 FROM public.vocabulary_topics vt
        WHERE vt.vocabulary_entry_id = v.id AND vt.topic_id = ANY(sqlc.arg(topic_ids))
    ))
ORDER BY
    CASE WHEN sqlc.arg(sort_order)::text = 'word_asc' THEN v.normalized_word_key END ASC,
    CASE WHEN sqlc.arg(sort_order)::text = 'word_desc' THEN v.normalized_word_key END DESC,
    CASE WHEN sqlc.arg(sort_order)::text = 'oldest' THEN v.created_at END ASC,
    v.created_at DESC, v.id DESC
LIMIT 10 OFFSET sqlc.arg(page_offset)::bigint;

-- name: VocabularyStats :one
SELECT count(*) AS total,
    count(*) FILTER (WHERE v.status = 'LEARNING') AS learning,
    count(*) FILTER (WHERE v.status = 'ACQUIRED') AS acquired,
    (SELECT count(DISTINCT r.library_item_id) FROM public.vocabulary_references r
        WHERE r.user_id = (SELECT id FROM public.users WHERE singleton)
            AND r.language_code = sqlc.arg(language_code)) AS referenced_texts
FROM public.vocabulary_entries v
WHERE v.user_id = (SELECT id FROM public.users WHERE singleton) AND v.language_code = sqlc.arg(language_code);

-- name: LockVocabularySelection :many
SELECT id FROM public.vocabulary_entries
WHERE user_id = (SELECT id FROM public.users WHERE singleton)
    AND language_code = sqlc.arg(language_code) AND id = ANY(sqlc.arg(ids)::uuid[])
ORDER BY id FOR UPDATE;

-- name: SetVocabularySelectionStatus :execrows
UPDATE public.vocabulary_entries SET status = sqlc.arg(status), updated_at = clock_timestamp()
WHERE user_id = (SELECT id FROM public.users WHERE singleton)
    AND language_code = sqlc.arg(language_code) AND id = ANY(sqlc.arg(ids)::uuid[]);

-- name: DeleteVocabularySelection :execrows
DELETE FROM public.vocabulary_entries
WHERE user_id = (SELECT id FROM public.users WHERE singleton)
    AND language_code = sqlc.arg(language_code) AND id = ANY(sqlc.arg(ids)::uuid[]);
