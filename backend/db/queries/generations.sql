-- name: GetOperationReceipt :one
SELECT operation, request_hash, generation_id, published_item_id, publication_error,
       (expires_at > statement_timestamp())::boolean AS replayable
FROM public.idempotency_records
WHERE user_id = sqlc.arg(user_id) AND key = sqlc.arg(key)
ORDER BY (operation = sqlc.arg(operation)) DESC
LIMIT 1;

-- name: ReserveGenerationKey :one
INSERT INTO public.idempotency_records (user_id, operation, key, request_hash, generation_id)
VALUES ($1, $2, $3, $4, gen_random_uuid())
RETURNING generation_id;

-- name: GenerationUsage :one
SELECT
    EXISTS (SELECT 1 FROM public.audio_generations g WHERE g.user_id = sqlc.arg(user_id)
            AND g.item_id = sqlc.arg(item_id) AND g.status IN ('PENDING', 'PROCESSING')) AS target_active,
    (SELECT count(*) FROM public.audio_generations g WHERE g.user_id = sqlc.arg(user_id)
     AND g.status IN ('PENDING', 'PROCESSING')) AS active_count,
    (SELECT count(*) FROM public.idempotency_records r WHERE r.user_id = sqlc.arg(user_id)
     AND r.operation IN ('generate_draft_audio', 'generate_preview_audio', 'generate_revision_audio')
     AND r.created_at > statement_timestamp() - interval '1 hour') AS hourly_count;

-- name: InsertGeneration :exec
INSERT INTO public.audio_generations (id, user_id, item_id, operation, idempotency_key, language_code, provider_id,
    voice_id, speaking_rate, strategy_id, strategy_version, fingerprint_format_version, source_text, source_fingerprint, expires_at, revision_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,
    CASE WHEN $4 = 'generate_preview_audio' THEN statement_timestamp() + interval '24 hours' END, $15);

-- name: LatestDraftCandidates :many
SELECT DISTINCT ON (g.item_id) g.item_id, g.id, g.source_fingerprint, g.status, g.error_code,
    a.id AS asset_id, a.storage_key, a.byte_size
FROM public.audio_generations g
LEFT JOIN public.audio_assets a ON a.id = g.audio_asset_id
    AND (a.language_code, a.provider_id, a.voice_id, a.speaking_rate, a.strategy_id, a.strategy_version, a.fingerprint_format_version)
      = (g.language_code, g.provider_id, g.voice_id, g.speaking_rate, g.strategy_id, g.strategy_version, g.fingerprint_format_version)
WHERE g.user_id = (SELECT u.id FROM public.users u WHERE u.singleton)
  AND g.item_id = ANY(sqlc.arg(item_ids)::uuid[]) AND g.revision_id IS NULL
ORDER BY g.item_id, g.admission_order DESC;

-- name: FindDraftGeneration :one
SELECT g.source_fingerprint, g.status, g.error_code
FROM public.audio_generations g
WHERE g.user_id = (SELECT u.id FROM public.users u WHERE u.singleton)
  AND g.item_id = $1 AND g.id = $2 AND g.revision_id IS NULL;
