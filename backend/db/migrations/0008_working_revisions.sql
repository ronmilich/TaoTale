CREATE TABLE public.working_revisions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    item_id uuid NOT NULL UNIQUE,
    base_audio_asset_id uuid NOT NULL,
    source_text text NOT NULL,
    voice_id text,
    speaking_rate double precision CHECK (speaking_rate IN (0.75, 1, 1.25)),
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (user_id, item_id) REFERENCES public.library_items (user_id, id) ON DELETE CASCADE,
    FOREIGN KEY (user_id, base_audio_asset_id) REFERENCES public.audio_assets (user_id, id),
    UNIQUE (user_id, item_id, id)
);

ALTER TABLE public.audio_generations
    ADD COLUMN revision_id uuid,
    ADD FOREIGN KEY (user_id, item_id, revision_id)
        REFERENCES public.working_revisions (user_id, item_id, id) ON DELETE CASCADE,
    DROP CONSTRAINT audio_generations_target_check,
    ADD CONSTRAINT audio_generations_target_check CHECK (
        (operation = 'generate_draft_audio' AND item_id IS NOT NULL AND revision_id IS NULL AND expires_at IS NULL)
        OR (operation = 'generate_preview_audio' AND revision_id IS NULL AND expires_at IS NOT NULL
            AND (item_id IS NULL OR status = 'READY'))
        OR (operation = 'generate_revision_audio' AND item_id IS NOT NULL AND revision_id IS NOT NULL AND expires_at IS NULL)
    );
CREATE TRIGGER audio_generations_revision_immutable BEFORE UPDATE OF revision_id ON public.audio_generations
    FOR EACH ROW EXECUTE FUNCTION public.reject_audio_snapshot_update();
CREATE INDEX audio_generations_revision_latest ON public.audio_generations (revision_id, admission_order DESC)
    WHERE revision_id IS NOT NULL;
