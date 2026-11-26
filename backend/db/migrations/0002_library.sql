CREATE TABLE public.library_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES public.users (id),
    language_code text NOT NULL REFERENCES public.learning_languages (code),
    title text CHECK (char_length(title) <= 120),
    source_text text NOT NULL DEFAULT '',
    description text CHECK (char_length(description) <= 1000),
    notes text CHECK (char_length(notes) <= 5000),
    difficulty text CHECK (difficulty IN ('beginner', 'intermediate', 'advanced')),
    is_favorite boolean NOT NULL DEFAULT false,
    voice_id text,
    speaking_rate double precision CHECK (speaking_rate IN (0.75, 1, 1.25)),
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    published_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, id)
);

CREATE INDEX library_items_recent ON public.library_items (user_id, language_code, created_at DESC, id DESC);
ALTER TABLE public.topics ADD UNIQUE (user_id, id);
ALTER TABLE public.topics ADD CHECK (char_length(name) <= 40);

CREATE TABLE public.item_topics (
    user_id uuid NOT NULL,
    item_id uuid NOT NULL,
    topic_id uuid NOT NULL,
    PRIMARY KEY (item_id, topic_id),
    FOREIGN KEY (user_id, item_id) REFERENCES public.library_items (user_id, id) ON DELETE CASCADE,
    FOREIGN KEY (user_id, topic_id) REFERENCES public.topics (user_id, id) ON DELETE CASCADE
);

CREATE INDEX item_topics_usage ON public.item_topics (topic_id, user_id);
