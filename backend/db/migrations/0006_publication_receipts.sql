-- Publication receipts survive deletion so a retry cannot create another item.
ALTER TABLE public.idempotency_records
    ALTER COLUMN generation_id DROP NOT NULL,
    ADD COLUMN published_item_id uuid,
    ADD COLUMN publication_error text,
    ADD CHECK (
        (operation = 'publish_preview' AND generation_id IS NULL
            AND num_nonnulls(published_item_id, publication_error) = 1)
        OR (operation <> 'publish_preview' AND generation_id IS NOT NULL
            AND published_item_id IS NULL AND publication_error IS NULL)
    );
