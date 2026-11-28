ALTER TABLE public.library_items ADD UNIQUE (user_id, language_code, id);

CREATE TABLE public.vocabulary_entries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES public.users (id),
    language_code text NOT NULL REFERENCES public.learning_languages (code),
    word text NOT NULL CHECK (char_length(word) BETWEEN 1 AND 120),
    normalized_word_key text COLLATE "C" NOT NULL CHECK (normalized_word_key <> ''),
    reading text CHECK (char_length(reading) <= 160),
    reading_system text CHECK (char_length(reading_system) <= 40),
    meaning text CHECK (char_length(meaning) <= 300),
    note text CHECK (char_length(note) <= 5000),
    status text NOT NULL DEFAULT 'LEARNING' CHECK (status IN ('LEARNING', 'ACQUIRED')),
    source_item_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT vocabulary_unique_word UNIQUE (user_id, language_code, normalized_word_key),
    UNIQUE (user_id, language_code, id),
    UNIQUE (user_id, id),
    FOREIGN KEY (user_id, language_code, source_item_id)
        REFERENCES public.library_items (user_id, language_code, id) ON DELETE SET NULL (source_item_id)
);

CREATE INDEX vocabulary_recent ON public.vocabulary_entries (user_id, language_code, created_at DESC, id DESC);
CREATE INDEX vocabulary_legacy_source ON public.vocabulary_entries (source_item_id) WHERE source_item_id IS NOT NULL;

CREATE TABLE public.vocabulary_references (
    user_id uuid NOT NULL,
    language_code text NOT NULL,
    vocabulary_entry_id uuid NOT NULL,
    library_item_id uuid NOT NULL,
    added_at timestamptz NOT NULL DEFAULT now(),
    manual_linked boolean NOT NULL DEFAULT false,
    auto_linked boolean NOT NULL DEFAULT false,
    PRIMARY KEY (vocabulary_entry_id, library_item_id),
    FOREIGN KEY (user_id, language_code, vocabulary_entry_id)
        REFERENCES public.vocabulary_entries (user_id, language_code, id) ON DELETE CASCADE,
    FOREIGN KEY (user_id, language_code, library_item_id)
        REFERENCES public.library_items (user_id, language_code, id) ON DELETE CASCADE,
    CHECK (manual_linked OR auto_linked)
);

CREATE INDEX vocabulary_references_item ON public.vocabulary_references (library_item_id, vocabulary_entry_id);
CREATE INDEX vocabulary_references_language ON public.vocabulary_references (user_id, language_code, library_item_id);

CREATE TABLE public.vocabulary_topics (
    user_id uuid NOT NULL,
    vocabulary_entry_id uuid NOT NULL,
    topic_id uuid NOT NULL,
    PRIMARY KEY (vocabulary_entry_id, topic_id),
    FOREIGN KEY (user_id, vocabulary_entry_id)
        REFERENCES public.vocabulary_entries (user_id, id) ON DELETE CASCADE,
    FOREIGN KEY (user_id, topic_id)
        REFERENCES public.topics (user_id, id) ON DELETE CASCADE
);

CREATE INDEX vocabulary_topics_topic ON public.vocabulary_topics (topic_id, user_id, vocabulary_entry_id);
