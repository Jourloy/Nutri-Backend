CREATE OR REPLACE FUNCTION rebrand_nutri_to_somivyn(input_text TEXT)
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
                                                        replace(input_text,
                                                            'https://api.nutri.jourloy.com', 'https://api.somivyn.com'
                                                        ),
                                                        'https://api-nutri.jourloy.com', 'https://api.somivyn.com'
                                                    ),
                                                    'https://nutri.jourloy.com', 'https://somivyn.com'
                                                ),
                                                'https://api.somivyn.jourloy.com', 'https://api.somivyn.com'
                                            ),
                                            'https://api-somivyn.jourloy.com', 'https://api.somivyn.com'
                                        ),
                                        'https://somivyn.jourloy.com', 'https://somivyn.com'
                                    ),
                                    'me-and-nutri', 'me-and-somivyn'
                                ),
                                'x-nutri-locale', 'x-somivyn-locale'
                            ),
                            'nutri-cta-link', 'somivyn-cta-link'
                        ),
                        'utm_source=nutri', 'utm_source=somivyn'
                    ),
                    'nutri-ai-images', 'somivyn-ai-images'
                ),
                'nutri-blog-images', 'somivyn-blog-images'
            ),
            'nutri-recipe-images', 'somivyn-recipe-images'
        ) AS value
    )
    SELECT CASE
        WHEN input_text IS NULL THEN NULL
        ELSE regexp_replace(
            regexp_replace(
                regexp_replace(
                    regexp_replace(
                        replace(value, 'nutri_', 'somivyn_'),
                        E'\\mНутри\\M', 'Somivyn', 'g'
                    ),
                    E'\\mнутри\\M', 'somivyn', 'g'
                ),
                E'\\mNutri\\M', 'Somivyn', 'g'
            ),
            E'\\mnutri\\M', 'somivyn', 'g'
        )
    END
    FROM replaced;
$$;

INSERT INTO translations (namespace, translation_key, locale, value, created_at, updated_at, deleted_at)
SELECT
    namespace,
    'dashboard.nav.meAndSomivyn',
    locale,
    rebrand_nutri_to_somivyn(value),
    created_at,
    NOW(),
    deleted_at
FROM translations
WHERE translation_key = 'dashboard.nav.meAndNutri'
ON CONFLICT (namespace, translation_key, locale) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW(),
    deleted_at = EXCLUDED.deleted_at;

DELETE FROM translations
WHERE translation_key = 'dashboard.nav.meAndNutri';

INSERT INTO translations (namespace, translation_key, locale, value, created_at, updated_at, deleted_at)
SELECT
    namespace,
    replace(translation_key, 'meAndNutri.', 'meAndSomivyn.'),
    locale,
    rebrand_nutri_to_somivyn(value),
    created_at,
    NOW(),
    deleted_at
FROM translations
WHERE translation_key LIKE 'meAndNutri.%'
ON CONFLICT (namespace, translation_key, locale) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW(),
    deleted_at = EXCLUDED.deleted_at;

DELETE FROM translations
WHERE translation_key LIKE 'meAndNutri.%';

INSERT INTO translations (namespace, translation_key, locale, value, created_at, updated_at, deleted_at)
SELECT
    namespace,
    replace(translation_key, 'home.nutriFeed.', 'home.somivynFeed.'),
    locale,
    rebrand_nutri_to_somivyn(value),
    created_at,
    NOW(),
    deleted_at
FROM translations
WHERE translation_key LIKE 'home.nutriFeed.%'
ON CONFLICT (namespace, translation_key, locale) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW(),
    deleted_at = EXCLUDED.deleted_at;

DELETE FROM translations
WHERE translation_key LIKE 'home.nutriFeed.%';

INSERT INTO translations (namespace, translation_key, locale, value, created_at, updated_at, deleted_at)
SELECT
    namespace,
    replace(translation_key, 'home.nutriRecipes.', 'home.somivynRecipes.'),
    locale,
    rebrand_nutri_to_somivyn(value),
    created_at,
    NOW(),
    deleted_at
FROM translations
WHERE translation_key LIKE 'home.nutriRecipes.%'
ON CONFLICT (namespace, translation_key, locale) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW(),
    deleted_at = EXCLUDED.deleted_at;

DELETE FROM translations
WHERE translation_key LIKE 'home.nutriRecipes.%';

UPDATE translations
SET value = rebrand_nutri_to_somivyn(value),
    updated_at = NOW()
WHERE value IS NOT NULL
  AND value <> rebrand_nutri_to_somivyn(value);

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
SET book_type = 'somivyn',
    name = rebrand_nutri_to_somivyn(name),
    og_image_url = rebrand_nutri_to_somivyn(og_image_url),
    updated_at = NOW()
WHERE book_type = 'nutri'
   OR name IS DISTINCT FROM rebrand_nutri_to_somivyn(name)
   OR og_image_url IS DISTINCT FROM rebrand_nutri_to_somivyn(og_image_url);

ALTER TABLE recipe_books
    ADD CONSTRAINT recipe_books_book_type_check
    CHECK (book_type IN ('somivyn', 'user'));

UPDATE blog_articles AS ba
SET title_ru = rebrand_nutri_to_somivyn(title_ru),
    title_en = rebrand_nutri_to_somivyn(title_en),
    content_ru = rebrand_nutri_to_somivyn(content_ru),
    content_en = rebrand_nutri_to_somivyn(content_en),
    preview_text_ru = rebrand_nutri_to_somivyn(preview_text_ru),
    preview_text_en = rebrand_nutri_to_somivyn(preview_text_en),
    preview_image_url = rebrand_nutri_to_somivyn(preview_image_url),
    meta_description_ru = rebrand_nutri_to_somivyn(meta_description_ru),
    meta_description_en = rebrand_nutri_to_somivyn(meta_description_en),
    og_image_url = rebrand_nutri_to_somivyn(og_image_url),
    canonical_url = rebrand_nutri_to_somivyn(canonical_url),
    sources = CASE
        WHEN ba.sources IS NULL THEN NULL
        ELSE ARRAY(
            SELECT rebrand_nutri_to_somivyn(source_item)
            FROM unnest(ba.sources) AS source_item
        )
    END,
    updated_at = NOW()
WHERE title_ru IS DISTINCT FROM rebrand_nutri_to_somivyn(title_ru)
   OR title_en IS DISTINCT FROM rebrand_nutri_to_somivyn(title_en)
   OR content_ru IS DISTINCT FROM rebrand_nutri_to_somivyn(content_ru)
   OR content_en IS DISTINCT FROM rebrand_nutri_to_somivyn(content_en)
   OR preview_text_ru IS DISTINCT FROM rebrand_nutri_to_somivyn(preview_text_ru)
   OR preview_text_en IS DISTINCT FROM rebrand_nutri_to_somivyn(preview_text_en)
   OR preview_image_url IS DISTINCT FROM rebrand_nutri_to_somivyn(preview_image_url)
   OR meta_description_ru IS DISTINCT FROM rebrand_nutri_to_somivyn(meta_description_ru)
   OR meta_description_en IS DISTINCT FROM rebrand_nutri_to_somivyn(meta_description_en)
   OR og_image_url IS DISTINCT FROM rebrand_nutri_to_somivyn(og_image_url)
   OR canonical_url IS DISTINCT FROM rebrand_nutri_to_somivyn(canonical_url)
   OR ba.sources IS DISTINCT FROM CASE
        WHEN ba.sources IS NULL THEN NULL
        ELSE ARRAY(
            SELECT rebrand_nutri_to_somivyn(source_item)
            FROM unnest(ba.sources) AS source_item
        )
    END;

UPDATE news
SET title_ru = rebrand_nutri_to_somivyn(title_ru),
    title_en = rebrand_nutri_to_somivyn(title_en),
    content_ru = rebrand_nutri_to_somivyn(content_ru),
    content_en = rebrand_nutri_to_somivyn(content_en),
    image_url = rebrand_nutri_to_somivyn(image_url),
    updated_at = NOW()
WHERE title_ru IS DISTINCT FROM rebrand_nutri_to_somivyn(title_ru)
   OR title_en IS DISTINCT FROM rebrand_nutri_to_somivyn(title_en)
   OR content_ru IS DISTINCT FROM rebrand_nutri_to_somivyn(content_ru)
   OR content_en IS DISTINCT FROM rebrand_nutri_to_somivyn(content_en)
   OR image_url IS DISTINCT FROM rebrand_nutri_to_somivyn(image_url);

UPDATE recipes
SET title_ru = rebrand_nutri_to_somivyn(title_ru),
    title_en = rebrand_nutri_to_somivyn(title_en),
    description_ru = rebrand_nutri_to_somivyn(description_ru),
    description_en = rebrand_nutri_to_somivyn(description_en),
    main_image_url = rebrand_nutri_to_somivyn(main_image_url),
    og_image_url = rebrand_nutri_to_somivyn(og_image_url),
    meta_description_ru = rebrand_nutri_to_somivyn(meta_description_ru),
    meta_description_en = rebrand_nutri_to_somivyn(meta_description_en),
    external_url = rebrand_nutri_to_somivyn(external_url),
    updated_at = NOW()
WHERE title_ru IS DISTINCT FROM rebrand_nutri_to_somivyn(title_ru)
   OR title_en IS DISTINCT FROM rebrand_nutri_to_somivyn(title_en)
   OR description_ru IS DISTINCT FROM rebrand_nutri_to_somivyn(description_ru)
   OR description_en IS DISTINCT FROM rebrand_nutri_to_somivyn(description_en)
   OR main_image_url IS DISTINCT FROM rebrand_nutri_to_somivyn(main_image_url)
   OR og_image_url IS DISTINCT FROM rebrand_nutri_to_somivyn(og_image_url)
   OR meta_description_ru IS DISTINCT FROM rebrand_nutri_to_somivyn(meta_description_ru)
   OR meta_description_en IS DISTINCT FROM rebrand_nutri_to_somivyn(meta_description_en)
   OR external_url IS DISTINCT FROM rebrand_nutri_to_somivyn(external_url);

UPDATE recipe_steps
SET instruction_ru = rebrand_nutri_to_somivyn(instruction_ru),
    instruction_en = rebrand_nutri_to_somivyn(instruction_en),
    image_url = rebrand_nutri_to_somivyn(image_url),
    updated_at = NOW()
WHERE instruction_ru IS DISTINCT FROM rebrand_nutri_to_somivyn(instruction_ru)
   OR instruction_en IS DISTINCT FROM rebrand_nutri_to_somivyn(instruction_en)
   OR image_url IS DISTINCT FROM rebrand_nutri_to_somivyn(image_url);

UPDATE recipe_images
SET image_url = rebrand_nutri_to_somivyn(image_url),
    caption_ru = rebrand_nutri_to_somivyn(caption_ru),
    caption_en = rebrand_nutri_to_somivyn(caption_en)
WHERE image_url IS DISTINCT FROM rebrand_nutri_to_somivyn(image_url)
   OR caption_ru IS DISTINCT FROM rebrand_nutri_to_somivyn(caption_ru)
   OR caption_en IS DISTINCT FROM rebrand_nutri_to_somivyn(caption_en);

DROP FUNCTION rebrand_nutri_to_somivyn(TEXT);
