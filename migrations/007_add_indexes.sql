-- +goose Up

CREATE INDEX idx_products_category_id
ON products(category_id);
CREATE INDEX idx_addresses_user_id
ON addresses(user_id);
CREATE INDEX idx_orders_user_id
ON orders(user_id);
CREATE INDEX idx_order_items_product_id
ON order_items(product_id);
CREATE INDEX idx_orders_status
ON orders(status);

-- +goose Down

DROP INDEX idx_orders_status;
DROP INDEX idx_order_items_product_id;
DROP INDEX idx_orders_user_id;
DROP INDEX idx_addresses_user_id;
DROP INDEX idx_products_category_id;
