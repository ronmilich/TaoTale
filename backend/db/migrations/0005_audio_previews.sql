ALTER TABLE public.audio_generations
    ALTER COLUMN item_id DROP NOT NULL,
    DROP CONSTRAINT audio_generations_operation_check,
    ADD COLUMN expires_at timestamptz,
    ADD CONSTRAINT audio_generations_target_check CHECK (
        (operation = 'generate_draft_audio' AND item_id IS NOT NULL AND expires_at IS NULL)
        OR (operation = 'generate_preview_audio' AND expires_at IS NOT NULL
            AND (item_id IS NULL OR status = 'READY'))
    );

DROP TRIGGER audio_generations_immutable ON public.audio_generations;
CREATE TRIGGER audio_generations_immutable BEFORE UPDATE OF
    id, user_id, operation, idempotency_key, language_code, provider_id, voice_id,
    speaking_rate, strategy_id, strategy_version, fingerprint_format_version,
    source_text, source_fingerprint, created_at ON public.audio_generations
    FOR EACH ROW EXECUTE FUNCTION public.reject_audio_snapshot_update();

CREATE FUNCTION public.allow_audio_preview_claim() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.operation = 'generate_preview_audio' AND OLD.item_id IS NULL
        AND NEW.item_id IS NOT NULL AND OLD.status = 'READY' AND OLD.audio_asset_id IS NOT NULL THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION 'Audio generation can only be attached once as a ready preview';
END;
$$;
CREATE TRIGGER audio_generations_claim_once BEFORE UPDATE OF item_id, admission_order
    ON public.audio_generations FOR EACH ROW EXECUTE FUNCTION public.allow_audio_preview_claim();

ALTER TABLE public.audio_assets ADD UNIQUE (user_id, id);
ALTER TABLE public.library_items
    ADD COLUMN active_audio_asset_id uuid,
    ADD FOREIGN KEY (user_id, active_audio_asset_id) REFERENCES public.audio_assets (user_id, id),
    ADD CHECK (active_audio_asset_id IS NULL OR published_at IS NOT NULL);
