-- name: SeedLocalOwner :exec
INSERT INTO public.users (id)
VALUES ('06dc9132-13d3-42da-a09c-000000000001')
ON CONFLICT (singleton) DO NOTHING;

-- name: GetLocalOwnerID :one
SELECT id FROM public.users WHERE singleton;

-- name: SeedLearningLanguages :exec
INSERT INTO public.learning_languages (code, display_name, enabled)
VALUES ('zh-CN', 'Mandarin Chinese', true), ('ko-KR', 'Korean', true)
ON CONFLICT (code) DO NOTHING;

-- name: SeedDefaultTopics :exec
INSERT INTO public.topics (
    id, user_id, name, name_key, system_key, light_background, light_text, dark_background, dark_text
)
SELECT defaults.id::uuid, sqlc.arg(user_id)::uuid, defaults.name, defaults.name_key, defaults.system_key,
       defaults.light_background, defaults.light_text, defaults.dark_background, defaults.dark_text
FROM (VALUES
    ('06dc9132-13d3-42da-a09c-000000000101', 'History',    'history',    'history',    '#FFF0DA', '#835921', '#54422B', '#F8DEB1'),
    ('06dc9132-13d3-42da-a09c-000000000102', 'Science',    'science',    'science',    '#E7EAFE', '#444D9E', '#31385C', '#DCE0FF'),
    ('06dc9132-13d3-42da-a09c-000000000103', 'Technology', 'technology', 'technology', '#ECE7FA', '#554095', '#3E345C', '#E4D8FF'),
    ('06dc9132-13d3-42da-a09c-000000000104', 'Business',   'business',   'business',   '#EDECF3', '#59566B', '#3A3749', '#DEDAED'),
    ('06dc9132-13d3-42da-a09c-000000000105', 'Culture',    'culture',    'culture',    '#F6E6F3', '#813E7B', '#513450', '#F4C8EC'),
    ('06dc9132-13d3-42da-a09c-000000000106', 'Daily Life', 'daily life', 'daily_life', '#ECE7FA', '#554095', '#3E345C', '#E4D8FF'),
    ('06dc9132-13d3-42da-a09c-000000000107', 'Travel',     'travel',     'travel',     '#E1F0E9', '#286449', '#294C3F', '#B9E9D2'),
    ('06dc9132-13d3-42da-a09c-000000000108', 'Health',     'health',     'health',     '#F6E6F3', '#813E7B', '#513450', '#F4C8EC')
) AS defaults(id, name, name_key, system_key, light_background, light_text, dark_background, dark_text)
WHERE NOT EXISTS (
    SELECT 1 FROM public.topics
    WHERE topics.user_id = sqlc.arg(user_id)::uuid AND topics.system_key = defaults.system_key
)
ON CONFLICT (user_id, system_key) WHERE system_key IS NOT NULL DO NOTHING;
