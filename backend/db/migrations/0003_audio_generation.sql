-- Each admission is also its permanent key tombstone and hourly cost entry.
-- The receipt is replayable until expires_at. Item deletion cannot free the key.
CREATE TABLE public.idempotency_records (
    user_id uuid NOT NULL REFERENCES public.users (id),
    operation text NOT NULL,
    key uuid NOT NULL,
    request_hash text NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    generation_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT statement_timestamp(),
    expires_at timestamptz NOT NULL DEFAULT statement_timestamp() + interval '24 hours',
    PRIMARY KEY (user_id, operation, key),
    UNIQUE (user_id, operation, key, generation_id)
);
CREATE INDEX idempotency_records_recent ON public.idempotency_records (user_id, operation, created_at);

-- An asset is inserted only after its MP3 and ordered segments pass validation.
CREATE TABLE public.audio_assets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES public.users (id),
    language_code text NOT NULL REFERENCES public.learning_languages (code),
    provider_id text NOT NULL,
    voice_id text NOT NULL,
    speaking_rate double precision NOT NULL CHECK (speaking_rate IN (0.75, 1, 1.25)),
    strategy_id text NOT NULL,
    strategy_version text NOT NULL,
    fingerprint_format_version integer NOT NULL CHECK (fingerprint_format_version > 0),
    source_fingerprint text NOT NULL CHECK (source_fingerprint ~ '^[0-9a-f]{64}$'),
    storage_key text NOT NULL UNIQUE,
    content_type text NOT NULL DEFAULT 'audio/mpeg' CHECK (content_type = 'audio/mpeg'),
    byte_size bigint NOT NULL CHECK (byte_size > 0),
    duration_ms bigint NOT NULL CHECK (duration_ms > 0),
    segments jsonb NOT NULL CHECK (jsonb_typeof(segments) = 'array' AND jsonb_array_length(segments) > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, id, source_fingerprint)
);

CREATE TABLE public.audio_generations (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    item_id uuid NOT NULL,
    operation text NOT NULL DEFAULT 'generate_draft_audio' CHECK (operation = 'generate_draft_audio'),
    idempotency_key uuid NOT NULL,
    admission_order bigint GENERATED ALWAYS AS IDENTITY,
    language_code text NOT NULL REFERENCES public.learning_languages (code),
    provider_id text NOT NULL,
    voice_id text NOT NULL,
    speaking_rate double precision NOT NULL CHECK (speaking_rate IN (0.75, 1, 1.25)),
    strategy_id text NOT NULL,
    strategy_version text NOT NULL,
    fingerprint_format_version integer NOT NULL CHECK (fingerprint_format_version > 0),
    source_text text NOT NULL,
    source_fingerprint text NOT NULL CHECK (source_fingerprint ~ '^[0-9a-f]{64}$'),
    status text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PROCESSING', 'READY', 'FAILED')),
    audio_asset_id uuid UNIQUE,
    error_code text,
    created_at timestamptz NOT NULL DEFAULT statement_timestamp(),
    FOREIGN KEY (user_id, item_id) REFERENCES public.library_items (user_id, id) ON DELETE CASCADE,
    FOREIGN KEY (user_id, operation, idempotency_key, id) REFERENCES public.idempotency_records (user_id, operation, key, generation_id),
    FOREIGN KEY (user_id, audio_asset_id, source_fingerprint) REFERENCES public.audio_assets (user_id, id, source_fingerprint),
    CHECK (audio_asset_id IS NULL OR status = 'READY'),
    CHECK ((status = 'FAILED') = (error_code IS NOT NULL))
);
CREATE UNIQUE INDEX audio_generations_active ON public.audio_generations (user_id, item_id)
    WHERE status IN ('PENDING', 'PROCESSING');
CREATE INDEX audio_generations_latest ON public.audio_generations (item_id, admission_order DESC);

CREATE FUNCTION public.reject_audio_snapshot_update() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'Audio synthesis snapshots are immutable';
END;
$$;
CREATE TRIGGER audio_generations_immutable BEFORE UPDATE OF
    id, user_id, item_id, operation, idempotency_key, admission_order, language_code, provider_id,
    voice_id, speaking_rate, strategy_id, strategy_version, fingerprint_format_version,
    source_text, source_fingerprint, created_at ON public.audio_generations
    FOR EACH ROW EXECUTE FUNCTION public.reject_audio_snapshot_update();
CREATE TRIGGER audio_assets_immutable BEFORE UPDATE ON public.audio_assets
    FOR EACH ROW EXECUTE FUNCTION public.reject_audio_snapshot_update();
