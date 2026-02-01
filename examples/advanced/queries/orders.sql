-- name: CreateOrder :one
INSERT INTO orders (user_id, status, total_amount)
VALUES (?, ?, ?)
RETURNING id, user_id, status, total_amount, created_at, updated_at;

-- name: GetOrder :one
SELECT 
    id,
    user_id,
    status,
    total_amount,
    created_at,
    updated_at
FROM orders
WHERE id = ?;

-- name: ListOrdersByUser :many
SELECT 
    id,
    status,
    total_amount,
    created_at
FROM orders
WHERE user_id = ?
ORDER BY created_at DESC;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = ?, updated_at = unixepoch()
WHERE id = ?
RETURNING id, user_id, status, total_amount, created_at, updated_at;

-- name: GetOrdersByStatus :many
SELECT 
    id,
    user_id,
    status,
    total_amount,
    created_at
FROM orders
WHERE status = ?
ORDER BY created_at DESC;

-- name: GetOrderStatistics :one
SELECT 
    COUNT(*) as total_orders,
    COALESCE(SUM(total_amount), 0) as total_revenue
FROM orders
WHERE user_id = ?;
