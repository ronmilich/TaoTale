-- name: CreateRevision :one
INSERT INTO public.working_revisions (user_id, item_id, base_audio_asset_id, source_text, voice_id, speaking_rate)
SELECT i.user_id, i.id, i.active_audio_asset_id, i.source_text, i.voice_id, i.speaking_rate
FROM public.library_items i WHERE i.id = $1 AND i.published_at IS NOT NULL
RETURNING *;

-- name: GetItemRevision :one
SELECT * FROM public.working_revisions WHERE item_id = $1 AND user_id = $2;

-- name: LockItemRevision :one
SELECT * FROM public.working_revisions WHERE item_id = $1 AND user_id = $2 FOR UPDATE;

-- name: UpdateRevision :one
UPDATE public.working_revisions SET source_text = $2, voice_id = $3, speaking_rate = $4,
    version = version + 1, updated_at = clock_timestamp() WHERE id = $1 AND version = $5
RETURNING *;

-- name: DeleteRevisionGenerations :many
DELETE FROM public.audio_generations WHERE revision_id = $1 RETURNING audio_asset_id;

-- name: DeleteRevision :exec
DELETE FROM public.working_revisions WHERE id = $1;

-- name: PublishRevision :execrows
UPDATE public.library_items SET source_text = $2, voice_id = $3, speaking_rate = $4,
    active_audio_asset_id = sqlc.arg(asset_id), version = version + 1, updated_at = clock_timestamp()
WHERE id = $1 AND version = sqlc.arg(expected_version) AND active_audio_asset_id = sqlc.arg(base_asset_id);

-- name: LatestRevisionCandidate :one
SELECT g.item_id, g.id, g.source_fingerprint, g.status, g.error_code,
    a.id AS asset_id, a.storage_key, a.byte_size
FROM public.audio_generations g
LEFT JOIN public.audio_assets a ON (a.user_id, a.id, a.source_fingerprint) = (g.user_id, g.audio_asset_id, g.source_fingerprint)
    AND (a.language_code, a.provider_id, a.voice_id, a.speaking_rate, a.strategy_id, a.strategy_version, a.fingerprint_format_version)
      = (g.language_code, g.provider_id, g.voice_id, g.speaking_rate, g.strategy_id, g.strategy_version, g.fingerprint_format_version)
WHERE g.revision_id = $1 ORDER BY g.admission_order DESC LIMIT 1;
