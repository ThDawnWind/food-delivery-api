-- +goose Up

CREATE TABLE product_images (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_product_images_product_id
ON product_images(product_id);

CREATE UNIQUE INDEX idx_product_images_one_primary
ON product_images(product_id)
WHERE is_primary = TRUE;

-- +goose Down

DROP TABLE product_images;
