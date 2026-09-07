-- +goose Up

CREATE TABLE orders (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'new' 
        CHECK (status IN (
            'new', 
            'confirmed', 
            'cooking', 
            'ready', 
            'delivering', 
            'completed', 
            'cancelled'
            )),
    total_price BIGINT NOT NULL CHECK (total_price > 0),
    delivery_address TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE orders;

