-- The caller holds the item lock, which also serializes generation admission
-- and preview claims. Select the latest admission before testing eligibility.
-- name: LockPublicationAudio :one
SELECT a.id, a.storage_key, a.byte_size, a.voice_id, a.speaking_rate
FROM public.audio_generations g
JOIN public.audio_assets a ON (a.user_id, a.id, a.source_fingerprint)
    = (g.user_id, g.audio_asset_id, g.source_fingerprint)
WHERE g.id = (
    SELECT latest.id FROM public.audio_generations latest
    WHERE latest.item_id = sqlc.arg(item_id) AND latest.user_id = sqlc.arg(user_id)
      AND latest.revision_id IS NOT DISTINCT FROM sqlc.narg(revision_id)::uuid
    ORDER BY latest.admission_order DESC LIMIT 1
)
  AND g.status = 'READY' AND g.source_text = sqlc.arg(source_text)
  AND g.source_fingerprint = sqlc.arg(source_fingerprint)
  AND (g.language_code, g.provider_id, g.voice_id, g.speaking_rate, g.strategy_id, g.strategy_version, g.fingerprint_format_version)
    = (a.language_code, a.provider_id, a.voice_id, a.speaking_rate, a.strategy_id, a.strategy_version, a.fingerprint_format_version)
  AND (a.language_code, a.provider_id, a.voice_id, a.speaking_rate, a.strategy_id, a.strategy_version, a.fingerprint_format_version)
    = (sqlc.arg(language_code), sqlc.arg(provider_id), sqlc.arg(voice_id)::text, sqlc.arg(speaking_rate)::float8,
       sqlc.arg(strategy_id), sqlc.arg(strategy_version), sqlc.arg(fingerprint_format_version)::int)
FOR SHARE OF g, a;

-- name: PublishItem :execrows
UPDATE public.library_items
SET published_at = statement_timestamp(), updated_at = statement_timestamp(), version = version + 1,
    active_audio_asset_id = sqlc.arg(asset_id), voice_id = sqlc.arg(voice_id), speaking_rate = sqlc.arg(speaking_rate)
WHERE library_items.id = sqlc.arg(item_id) AND published_at IS NULL AND version = sqlc.arg(expected_version)
  AND user_id = (SELECT u.id FROM public.users u WHERE u.singleton);

-- name: RecordPublication :exec
INSERT INTO public.idempotency_records (user_id, operation, key, request_hash, published_item_id, publication_error)
VALUES (sqlc.arg(user_id), 'publish_preview', sqlc.arg(key), sqlc.arg(request_hash),
        sqlc.narg(published_item_id), sqlc.narg(publication_error));
