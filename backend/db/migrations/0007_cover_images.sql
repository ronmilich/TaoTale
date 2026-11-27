CREATE TABLE public.media_images (
    id uuid PRIMARY KEY,
    owner_id uuid NOT NULL REFERENCES public.users (id),
    storage_key text NOT NULL UNIQUE,
    content_type text NOT NULL CHECK (content_type IN ('image/jpeg', 'image/png', 'image/webp')),
    size_bytes bigint NOT NULL CHECK (size_bytes > 0),
    width integer NOT NULL CHECK (width > 0),
    height integer NOT NULL CHECK (height > 0),
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (owner_id, id)
);

ALTER TABLE public.library_items ADD COLUMN cover_image_id uuid;
ALTER TABLE public.library_items ADD FOREIGN KEY (user_id, cover_image_id)
    REFERENCES public.media_images (owner_id, id);
