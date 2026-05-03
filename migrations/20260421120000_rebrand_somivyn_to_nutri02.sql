CREATE OR REPLACE FUNCTION rebrand_somivyn_to_nutri02(input_text TEXT)
RETURNS TEXT
LANGUAGE SQL
IMMUTABLE
AS $$
    WITH replaced AS (
        SELECT replace(
            replace(
                replace(
                    replace(
                        replace(
                            replace(
                                replace(
                                    replace(
                                        replace(
                                            replace(
                                                replace(
                                                    replace(
                                                        replace(
                                                            replace(
                                                                replace(
                                                                    replace(input_text,
                                                                        'support@jourloy.com', 'hello@nutri02.com'
                                                                    ),
                                                                    'hello@somivyn.com', 'hello@nutri02.com'
                                                                ),
                                                                'noreply@somivyn.com', 'noreply@nutri02.com'
                                                            ),
                                                            'https://api.somivyn.com', 'https://api.nutri02.com'
                                                        ),
                                                        'https://api.somivyn.jourloy.com', 'https://api.nutri02.com'
                                                    ),
                                                    'https://api-somivyn.jourloy.com', 'https://api.nutri02.com'
                                                ),
                                                'https://api.nutri.jourloy.com', 'https://api.nutri02.com'
                                            ),
                                            'https://api-nutri.jourloy.com', 'https://api.nutri02.com'
                                        ),
                                        'https://somivyn.com', 'https://nutri02.com'
                                    ),
                                    'https://somivyn.jourloy.com', 'https://nutri02.com'
                                ),
                                'https://nutri.jourloy.com', 'https://nutri02.com'
                            ),
                            'https://nutri02.jourloy.com', 'https://nutri02.com'
                        ),
                        'https://s3.somivyn.com', 'https://s3.nutri02.com'
                    ),
                    'me-and-somivyn', 'me-and-nutri02'
                ),
                'x-somivyn-locale', 'x-nutri02-locale'
            ),
            'somivyn-cta-link', 'nutri02-cta-link'
        ) AS value
    )
    SELECT CASE
        WHEN input_text IS NULL THEN NULL
        ELSE regexp_replace(
            regexp_replace(
                regexp_replace(
                    regexp_replace(
                        regexp_replace(
                            regexp_replace(
                                regexp_replace(
                                    replace(
                                        replace(
                                            replace(
                                                replace(value, 'utm_source=somivyn', 'utm_source=nutri02'),
                                                'somivyn-blog-content', 'nutri02-blog-content'
                                            ),
                                            'somivyn-ai-images', 'nutri02-ai-images'
                                        ),
                                        'somivyn-blog-images', 'nutri02-blog-images'
                                    ),
                                    'somivyn-recipe-images', 'nutri02-recipe-images'
                                ),
                                'somivyn_', 'nutri02_'
                            ),
                            E'\\mSomivyn\\M', 'Nutri02', 'g'
                        ),
                        E'\\msomivyn\\M', 'nutri02', 'g'
                    ),
                    E'\\mСомивин\\M', 'Nutri02', 'g'
                ),
                E'\\mсомивин\\M', 'nutri02', 'g'
            ),
            E'\\mSomivyn\\M', 'Nutri02', 'g'
        )
    END
    FROM replaced;
$$;

INSERT INTO translations (namespace, translation_key, locale, value, created_at, updated_at, deleted_at)
SELECT
    namespace,
    'dashboard.nav.meAndNutri02',
    locale,
    rebrand_somivyn_to_nutri02(value),
    created_at,
    NOW(),
    deleted_at
FROM translations
WHERE translation_key = 'dashboard.nav.meAndSomivyn'
ON CONFLICT (namespace, translation_key, locale) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW(),
    deleted_at = EXCLUDED.deleted_at;

DELETE FROM translations
WHERE translation_key = 'dashboard.nav.meAndSomivyn';

INSERT INTO translations (namespace, translation_key, locale, value, created_at, updated_at, deleted_at)
SELECT
    namespace,
    replace(translation_key, 'meAndSomivyn.', 'meAndNutri02.'),
    locale,
    rebrand_somivyn_to_nutri02(value),
    created_at,
    NOW(),
    deleted_at
FROM translations
WHERE translation_key LIKE 'meAndSomivyn.%'
ON CONFLICT (namespace, translation_key, locale) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW(),
    deleted_at = EXCLUDED.deleted_at;

DELETE FROM translations
WHERE translation_key LIKE 'meAndSomivyn.%';

INSERT INTO translations (namespace, translation_key, locale, value, created_at, updated_at, deleted_at)
SELECT
    namespace,
    replace(translation_key, 'home.somivynFeed.', 'home.nutri02Feed.'),
    locale,
    rebrand_somivyn_to_nutri02(value),
    created_at,
    NOW(),
    deleted_at
FROM translations
WHERE translation_key LIKE 'home.somivynFeed.%'
ON CONFLICT (namespace, translation_key, locale) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW(),
    deleted_at = EXCLUDED.deleted_at;

DELETE FROM translations
WHERE translation_key LIKE 'home.somivynFeed.%';

INSERT INTO translations (namespace, translation_key, locale, value, created_at, updated_at, deleted_at)
SELECT
    namespace,
    replace(translation_key, 'home.somivynRecipes.', 'home.nutri02Recipes.'),
    locale,
    rebrand_somivyn_to_nutri02(value),
    created_at,
    NOW(),
    deleted_at
FROM translations
WHERE translation_key LIKE 'home.somivynRecipes.%'
ON CONFLICT (namespace, translation_key, locale) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW(),
    deleted_at = EXCLUDED.deleted_at;

DELETE FROM translations
WHERE translation_key LIKE 'home.somivynRecipes.%';

UPDATE translations
SET value = rebrand_somivyn_to_nutri02(value),
    updated_at = NOW()
WHERE value IS NOT NULL
  AND value <> rebrand_somivyn_to_nutri02(value);

DO $$
DECLARE
    constraint_name TEXT;
BEGIN
    SELECT conname
    INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'recipe_books'::regclass
      AND contype = 'c'
      AND pg_get_constraintdef(oid) ILIKE '%book_type%';

    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE recipe_books DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;

UPDATE recipe_books
SET book_type = 'nutri02',
    name = rebrand_somivyn_to_nutri02(name),
    og_image_url = rebrand_somivyn_to_nutri02(og_image_url),
    updated_at = NOW()
WHERE book_type IN ('somivyn', 'nutri')
   OR name IS DISTINCT FROM rebrand_somivyn_to_nutri02(name)
   OR og_image_url IS DISTINCT FROM rebrand_somivyn_to_nutri02(og_image_url);

ALTER TABLE recipe_books
    ADD CONSTRAINT recipe_books_book_type_check
    CHECK (book_type IN ('nutri02', 'user'));

UPDATE blog_articles AS ba
SET title_ru = rebrand_somivyn_to_nutri02(title_ru),
    title_en = rebrand_somivyn_to_nutri02(title_en),
    content_ru = rebrand_somivyn_to_nutri02(content_ru),
    content_en = rebrand_somivyn_to_nutri02(content_en),
    preview_text_ru = rebrand_somivyn_to_nutri02(preview_text_ru),
    preview_text_en = rebrand_somivyn_to_nutri02(preview_text_en),
    preview_image_url = rebrand_somivyn_to_nutri02(preview_image_url),
    meta_description_ru = rebrand_somivyn_to_nutri02(meta_description_ru),
    meta_description_en = rebrand_somivyn_to_nutri02(meta_description_en),
    og_image_url = rebrand_somivyn_to_nutri02(og_image_url),
    canonical_url = rebrand_somivyn_to_nutri02(canonical_url),
    sources = CASE
        WHEN ba.sources IS NULL THEN NULL
        ELSE ARRAY(
            SELECT rebrand_somivyn_to_nutri02(source_item)
            FROM unnest(ba.sources) AS source_item
        )
    END,
    updated_at = NOW()
WHERE title_ru IS DISTINCT FROM rebrand_somivyn_to_nutri02(title_ru)
   OR title_en IS DISTINCT FROM rebrand_somivyn_to_nutri02(title_en)
   OR content_ru IS DISTINCT FROM rebrand_somivyn_to_nutri02(content_ru)
   OR content_en IS DISTINCT FROM rebrand_somivyn_to_nutri02(content_en)
   OR preview_text_ru IS DISTINCT FROM rebrand_somivyn_to_nutri02(preview_text_ru)
   OR preview_text_en IS DISTINCT FROM rebrand_somivyn_to_nutri02(preview_text_en)
   OR preview_image_url IS DISTINCT FROM rebrand_somivyn_to_nutri02(preview_image_url)
   OR meta_description_ru IS DISTINCT FROM rebrand_somivyn_to_nutri02(meta_description_ru)
   OR meta_description_en IS DISTINCT FROM rebrand_somivyn_to_nutri02(meta_description_en)
   OR og_image_url IS DISTINCT FROM rebrand_somivyn_to_nutri02(og_image_url)
   OR canonical_url IS DISTINCT FROM rebrand_somivyn_to_nutri02(canonical_url)
   OR ba.sources IS DISTINCT FROM CASE
        WHEN ba.sources IS NULL THEN NULL
        ELSE ARRAY(
            SELECT rebrand_somivyn_to_nutri02(source_item)
            FROM unnest(ba.sources) AS source_item
        )
    END;

UPDATE news
SET title_ru = rebrand_somivyn_to_nutri02(title_ru),
    title_en = rebrand_somivyn_to_nutri02(title_en),
    content_ru = rebrand_somivyn_to_nutri02(content_ru),
    content_en = rebrand_somivyn_to_nutri02(content_en),
    image_url = rebrand_somivyn_to_nutri02(image_url),
    updated_at = NOW()
WHERE title_ru IS DISTINCT FROM rebrand_somivyn_to_nutri02(title_ru)
   OR title_en IS DISTINCT FROM rebrand_somivyn_to_nutri02(title_en)
   OR content_ru IS DISTINCT FROM rebrand_somivyn_to_nutri02(content_ru)
   OR content_en IS DISTINCT FROM rebrand_somivyn_to_nutri02(content_en)
   OR image_url IS DISTINCT FROM rebrand_somivyn_to_nutri02(image_url);

UPDATE recipes
SET title_ru = rebrand_somivyn_to_nutri02(title_ru),
    title_en = rebrand_somivyn_to_nutri02(title_en),
    description_ru = rebrand_somivyn_to_nutri02(description_ru),
    description_en = rebrand_somivyn_to_nutri02(description_en),
    main_image_url = rebrand_somivyn_to_nutri02(main_image_url),
    og_image_url = rebrand_somivyn_to_nutri02(og_image_url),
    meta_description_ru = rebrand_somivyn_to_nutri02(meta_description_ru),
    meta_description_en = rebrand_somivyn_to_nutri02(meta_description_en),
    external_url = rebrand_somivyn_to_nutri02(external_url),
    updated_at = NOW()
WHERE title_ru IS DISTINCT FROM rebrand_somivyn_to_nutri02(title_ru)
   OR title_en IS DISTINCT FROM rebrand_somivyn_to_nutri02(title_en)
   OR description_ru IS DISTINCT FROM rebrand_somivyn_to_nutri02(description_ru)
   OR description_en IS DISTINCT FROM rebrand_somivyn_to_nutri02(description_en)
   OR main_image_url IS DISTINCT FROM rebrand_somivyn_to_nutri02(main_image_url)
   OR og_image_url IS DISTINCT FROM rebrand_somivyn_to_nutri02(og_image_url)
   OR meta_description_ru IS DISTINCT FROM rebrand_somivyn_to_nutri02(meta_description_ru)
   OR meta_description_en IS DISTINCT FROM rebrand_somivyn_to_nutri02(meta_description_en)
   OR external_url IS DISTINCT FROM rebrand_somivyn_to_nutri02(external_url);

UPDATE recipe_steps
SET instruction_ru = rebrand_somivyn_to_nutri02(instruction_ru),
    instruction_en = rebrand_somivyn_to_nutri02(instruction_en),
    image_url = rebrand_somivyn_to_nutri02(image_url),
    updated_at = NOW()
WHERE instruction_ru IS DISTINCT FROM rebrand_somivyn_to_nutri02(instruction_ru)
   OR instruction_en IS DISTINCT FROM rebrand_somivyn_to_nutri02(instruction_en)
   OR image_url IS DISTINCT FROM rebrand_somivyn_to_nutri02(image_url);

UPDATE recipe_images
SET image_url = rebrand_somivyn_to_nutri02(image_url),
    caption_ru = rebrand_somivyn_to_nutri02(caption_ru),
    caption_en = rebrand_somivyn_to_nutri02(caption_en)
WHERE image_url IS DISTINCT FROM rebrand_somivyn_to_nutri02(image_url)
   OR caption_ru IS DISTINCT FROM rebrand_somivyn_to_nutri02(caption_ru)
   OR caption_en IS DISTINCT FROM rebrand_somivyn_to_nutri02(caption_en);

DROP FUNCTION rebrand_somivyn_to_nutri02(TEXT);
