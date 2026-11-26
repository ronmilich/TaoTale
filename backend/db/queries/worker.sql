-- name: AcquireWorkerLeadership :one
SELECT pg_try_advisory_lock(843672910197::bigint)::boolean;

-- name: InterruptUnfinishedJobs :exec
UPDATE public.audio_generations
SET status = 'FAILED', error_code = 'GENERATION_INTERRUPTED', completed_at = statement_timestamp()
WHERE status IN ('PENDING', 'PROCESSING');

-- name: ClaimGeneration :one
UPDATE public.audio_generations
SET status = 'PROCESSING', started_at = statement_timestamp(), heartbeat_at = statement_timestamp()
WHERE id = (SELECT g.id FROM public.audio_generations g WHERE g.status = 'PENDING'
            ORDER BY g.admission_order LIMIT 1 FOR UPDATE SKIP LOCKED)
RETURNING *;

-- name: HeartbeatGeneration :execrows
UPDATE public.audio_generations SET heartbeat_at = statement_timestamp() WHERE id = $1 AND status = 'PROCESSING';

-- name: FailGeneration :exec
UPDATE public.audio_generations SET status = 'FAILED', error_code = sqlc.arg(code)::text, completed_at = statement_timestamp()
WHERE id = $1 AND status = 'PROCESSING';

-- name: LockGenerationForAsset :one
SELECT gen_random_uuid()::uuid AS asset_id FROM public.audio_generations
WHERE id = $1 AND status = 'PROCESSING' FOR UPDATE;

-- name: InsertVerifiedAsset :exec
INSERT INTO public.audio_assets (id, user_id, language_code, provider_id, voice_id, speaking_rate, strategy_id, strategy_version,
    fingerprint_format_version, source_fingerprint, storage_key, byte_size, duration_ms, segments)
SELECT sqlc.arg(asset_id)::uuid, user_id, language_code, provider_id, voice_id, speaking_rate, strategy_id, strategy_version,
    fingerprint_format_version, source_fingerprint, sqlc.arg(storage_key)::text, sqlc.arg(byte_size)::bigint,
    sqlc.arg(duration_ms)::bigint, sqlc.arg(segments)::jsonb
FROM public.audio_generations g WHERE g.id = sqlc.arg(generation_id);

-- name: FinishGeneration :exec
UPDATE public.audio_generations SET status = 'READY', audio_asset_id = sqlc.arg(asset_id)::uuid, completed_at = statement_timestamp()
WHERE id = $1 AND status = 'PROCESSING';

-- name: ReferencedAudioFiles :many
SELECT storage_key FROM public.audio_assets WHERE storage_key = ANY(sqlc.arg(keys)::text[]);
