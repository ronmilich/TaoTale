-- name: LockMatchingVocabulary :many
SELECT v.id FROM public.vocabulary_entries v
WHERE v.user_id = sqlc.arg(user_id) AND v.language_code = sqlc.arg(language_code)
    AND strpos(sqlc.arg(source_text)::text COLLATE "C", v.word COLLATE "C") > 0
ORDER BY v.id
FOR SHARE;

-- name: RemoveObsoleteAutomaticVocabulary :exec
WITH obsolete_automatic_only AS (
    DELETE FROM public.vocabulary_references r
    WHERE r.user_id = sqlc.arg(user_id) AND r.library_item_id = sqlc.arg(item_id)
        AND r.auto_linked AND NOT r.manual_linked
        AND NOT (r.vocabulary_entry_id = ANY(coalesce(sqlc.arg(matching_ids)::uuid[], '{}')))
)
UPDATE public.vocabulary_references r
SET auto_linked = false
WHERE r.user_id = sqlc.arg(user_id) AND r.library_item_id = sqlc.arg(item_id)
    AND r.auto_linked AND r.manual_linked
    AND NOT (r.vocabulary_entry_id = ANY(coalesce(sqlc.arg(matching_ids)::uuid[], '{}')));

-- name: AddAutomaticVocabulary :exec
INSERT INTO public.vocabulary_references
    (user_id, language_code, vocabulary_entry_id, library_item_id, auto_linked)
SELECT sqlc.arg(user_id)::uuid, sqlc.arg(language_code)::text, word.id, sqlc.arg(item_id)::uuid, true
FROM unnest(sqlc.arg(matching_ids)::uuid[]) AS word(id)
ON CONFLICT (vocabulary_entry_id, library_item_id)
DO UPDATE SET auto_linked = true WHERE NOT vocabulary_references.auto_linked;
