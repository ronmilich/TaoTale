CREATE TABLE public.users (
    id uuid PRIMARY KEY,
    singleton boolean NOT NULL DEFAULT true UNIQUE CHECK (singleton),
    display_name text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.learning_languages (
    code text PRIMARY KEY,
    display_name text NOT NULL CHECK (btrim(display_name) <> ''),
    enabled boolean NOT NULL
);

CREATE TABLE public.topics (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES public.users (id),
    name text NOT NULL CHECK (btrim(name) <> ''),
    name_key text COLLATE "C" NOT NULL CHECK (btrim(name_key) <> ''),
    system_key text COLLATE "C" CHECK (system_key ~ '^[a-z][a-z0-9_]*$'),
    light_background text NOT NULL CHECK (light_background ~ '^#[0-9a-fA-F]{6}$'),
    light_text text NOT NULL CHECK (light_text ~ '^#[0-9a-fA-F]{6}$'),
    dark_background text NOT NULL CHECK (dark_background ~ '^#[0-9a-fA-F]{6}$'),
    dark_text text NOT NULL CHECK (dark_text ~ '^#[0-9a-fA-F]{6}$'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, name_key)
);

CREATE UNIQUE INDEX topics_user_system_key ON public.topics (user_id, system_key)
    WHERE system_key IS NOT NULL;

CREATE FUNCTION public.reject_default_topic_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'default topic is protected' USING ERRCODE = '23514';
END;
$$;

CREATE TRIGGER protect_default_topic_delete
    BEFORE DELETE ON public.topics
    FOR EACH ROW WHEN (OLD.system_key IS NOT NULL)
    EXECUTE FUNCTION public.reject_default_topic_mutation();

CREATE TRIGGER protect_default_topic_key
    BEFORE UPDATE OF system_key ON public.topics
    FOR EACH ROW WHEN (OLD.system_key IS NOT NULL AND NEW.system_key IS DISTINCT FROM OLD.system_key)
    EXECUTE FUNCTION public.reject_default_topic_mutation();
