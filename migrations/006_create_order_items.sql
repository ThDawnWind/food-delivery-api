-- +goose Up

CREATE TABLE order_items (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products(id),
    name_snapshot VARCHAR(150) NOT NULL,
    price_snapshot BIGINT NOT NULL CHECK (price_snapshot > 0),
    quantity INT NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (order_id, product_id)
);

-- +goose Down
DROP TABLE order_items;

