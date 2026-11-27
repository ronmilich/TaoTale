-- name: InsertCoverImage :one
INSERT INTO public.media_images (id, owner_id, storage_key, content_type, size_bytes, width, height, expires_at)
SELECT sqlc.arg(id), u.id, sqlc.arg(storage_key), sqlc.arg(content_type), sqlc.arg(size_bytes),
       sqlc.arg(width), sqlc.arg(height), now() + interval '24 hours'
FROM public.users u WHERE u.singleton
RETURNING *;

-- name: GetCoverImage :one
SELECT * FROM public.media_images
WHERE media_images.id = $1 AND owner_id = (SELECT u.id FROM public.users u WHERE u.singleton)
    AND (expires_at IS NULL OR expires_at > now());

-- name: ClaimCoverImage :one
UPDATE public.media_images SET expires_at = NULL
WHERE media_images.id = $1 AND owner_id = (SELECT u.id FROM public.users u WHERE u.singleton)
    AND (expires_at IS NULL OR expires_at > now())
RETURNING *;
