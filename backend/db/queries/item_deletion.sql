-- The item is already locked. Deleting the jobs here waits for any worker
-- finishing a job and returns even the asset produced by that worker.
-- name: DeleteItemGenerations :many
DELETE FROM public.audio_generations
WHERE user_id = $1 AND item_id = $2
RETURNING audio_asset_id;

-- Check references in a fresh statement after the locks are acquired, so a
-- concurrent delete of another item sharing media can release the last copy.
-- name: LockDeletionAudio :exec
SELECT id FROM public.audio_assets WHERE user_id = $1 AND id = ANY(sqlc.arg(asset_ids)::uuid[])
ORDER BY id FOR UPDATE;

-- name: DeleteUnreferencedAudio :many
DELETE FROM public.audio_assets a
WHERE a.user_id = $1 AND a.id = ANY(sqlc.arg(asset_ids)::uuid[])
  AND NOT EXISTS (SELECT 1 FROM public.library_items i WHERE (i.user_id, i.active_audio_asset_id) = (a.user_id, a.id))
  AND NOT EXISTS (SELECT 1 FROM public.working_revisions r WHERE (r.user_id, r.base_audio_asset_id) = (a.user_id, a.id))
  AND NOT EXISTS (SELECT 1 FROM public.audio_generations g WHERE (g.user_id, g.audio_asset_id) = (a.user_id, a.id))
RETURNING storage_key;

-- name: LockDeletionCover :exec
SELECT id FROM public.media_images WHERE owner_id = $1 AND id = $2 FOR UPDATE;

-- name: DeleteUnreferencedCover :one
DELETE FROM public.media_images m
WHERE m.owner_id = $1 AND m.id = $2
  AND NOT EXISTS (SELECT 1 FROM public.library_items i WHERE (i.user_id, i.cover_image_id) = (m.owner_id, m.id))
RETURNING storage_key;
