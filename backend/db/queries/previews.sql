-- name: FindPreview :one
SELECT g.id, g.status, g.error_code, g.source_fingerprint, g.expires_at,
    a.id AS asset_id, a.storage_key, a.byte_size
FROM public.audio_generations g
LEFT JOIN public.audio_assets a ON a.id = g.audio_asset_id
    AND (a.language_code, a.provider_id, a.voice_id, a.speaking_rate, a.strategy_id, a.strategy_version, a.fingerprint_format_version)
      = (g.language_code, g.provider_id, g.voice_id, g.speaking_rate, g.strategy_id, g.strategy_version, g.fingerprint_format_version)
WHERE g.user_id = (SELECT u.id FROM public.users u WHERE u.singleton)
  AND g.id = $1 AND g.operation = 'generate_preview_audio' AND g.item_id IS NULL AND g.expires_at > statement_timestamp();

-- name: AttachPreview :execrows
UPDATE public.audio_generations g
SET item_id = sqlc.arg(item_id), admission_order = DEFAULT
WHERE g.user_id = (SELECT u.id FROM public.users u WHERE u.singleton)
  AND g.id = sqlc.arg(id) AND operation = 'generate_preview_audio' AND g.item_id IS NULL
  AND status = 'READY' AND audio_asset_id IS NOT NULL AND expires_at > clock_timestamp();

-- name: PublishedAudioAssets :many
SELECT i.id AS item_id, a.id AS asset_id, a.storage_key, a.byte_size
FROM public.library_items i
JOIN public.audio_assets a ON a.id = i.active_audio_asset_id AND a.user_id = i.user_id
  AND a.language_code = i.language_code
WHERE i.user_id = (SELECT u.id FROM public.users u WHERE u.singleton)
  AND i.published_at IS NOT NULL AND i.id = ANY(sqlc.arg(item_ids)::uuid[]);
