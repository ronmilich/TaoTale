ALTER TABLE public.audio_generations
    ADD COLUMN started_at timestamptz,
    ADD COLUMN heartbeat_at timestamptz,
    ADD COLUMN completed_at timestamptz;
CREATE INDEX audio_generations_pending ON public.audio_generations (admission_order)
    WHERE status = 'PENDING';
