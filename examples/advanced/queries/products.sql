-- name: CreateProduct :one
INSERT INTO products (sku, name, price)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetProduct :one
SELECT * FROM products WHERE id = ?;

-- name: GetProductBySKU :one
SELECT * FROM products WHERE sku = ?;

-- name: ListProducts :many
SELECT * FROM products ORDER BY name;

-- name: UpdateProductPrice :one
UPDATE products
SET price = ?
WHERE id = ?
RETURNING *;
