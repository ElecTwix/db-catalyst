-- SummarizeCredits aggregates user credits across a recursive rollup.
-- name: SummarizeCredits :one
WITH RECURSIVE credit_totals AS (
    SELECT id, credits FROM users
    UNION ALL
    SELECT u.id, u.credits FROM users u JOIN credit_totals c ON u.id > c.id
)
SELECT COUNT(*) AS total_users,
       SUM(credit_totals.credits) AS sum_credits,
       AVG(credit_totals.credits) AS avg_credit
FROM credit_totals;
