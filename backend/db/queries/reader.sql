-- name: GetTranscript :one
SELECT duration_ms, segments FROM public.audio_assets
WHERE user_id = sqlc.arg(user_id) AND id = sqlc.arg(asset_id);
