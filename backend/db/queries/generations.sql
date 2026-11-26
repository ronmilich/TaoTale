-- name: GetGenerationReceipt :one
SELECT request_hash, generation_id, (expires_at > statement_timestamp())::boolean AS replayable
FROM public.idempotency_records
WHERE user_id = $1 AND operation = 'generate_draft_audio' AND key = $2;

-- name: ReserveGenerationKey :one
INSERT INTO public.idempotency_records (user_id, operation, key, request_hash, generation_id)
VALUES ($1, 'generate_draft_audio', $2, $3, gen_random_uuid())
RETURNING generation_id;

-- name: GenerationUsage :one
SELECT
    EXISTS (SELECT 1 FROM public.audio_generations g WHERE g.user_id = sqlc.arg(user_id)
            AND g.item_id = sqlc.arg(item_id) AND g.status IN ('PENDING', 'PROCESSING')) AS target_active,
    (SELECT count(*) FROM public.audio_generations g WHERE g.user_id = sqlc.arg(user_id)
     AND g.status IN ('PENDING', 'PROCESSING')) AS active_count,
    (SELECT count(*) FROM public.idempotency_records r WHERE r.user_id = sqlc.arg(user_id)
     AND r.operation = 'generate_draft_audio' AND r.created_at > statement_timestamp() - interval '1 hour') AS hourly_count;

-- name: InsertGeneration :exec
INSERT INTO public.audio_generations (id, user_id, item_id, idempotency_key, language_code, provider_id,
    voice_id, speaking_rate, strategy_id, strategy_version, fingerprint_format_version, source_text, source_fingerprint)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13);

-- name: LatestDraftCandidates :many
SELECT DISTINCT ON (g.item_id) g.item_id, g.id, g.source_fingerprint, g.status, g.error_code,
    a.id AS asset_id, a.storage_key, a.byte_size
FROM public.audio_generations g
LEFT JOIN public.audio_assets a ON a.id = g.audio_asset_id
    AND (a.language_code, a.provider_id, a.voice_id, a.speaking_rate, a.strategy_id, a.strategy_version, a.fingerprint_format_version)
      = (g.language_code, g.provider_id, g.voice_id, g.speaking_rate, g.strategy_id, g.strategy_version, g.fingerprint_format_version)
WHERE g.user_id = (SELECT u.id FROM public.users u WHERE u.singleton)
  AND g.item_id = ANY(sqlc.arg(item_ids)::uuid[])
ORDER BY g.item_id, g.admission_order DESC;

-- name: FindDraftGeneration :one
SELECT g.source_fingerprint, g.status, g.error_code
FROM public.audio_generations g
WHERE g.user_id = (SELECT u.id FROM public.users u WHERE u.singleton)
  AND g.item_id = $1 AND g.id = $2;
