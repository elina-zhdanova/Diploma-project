-- name: InsertAIDecision :one
INSERT INTO ai_decisions (request_id, recommendation, risk_score, confidence, reason)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, request_id, recommendation, risk_score, confidence, reason, created_at;

-- name: ListAIDecisionsByRequest :many
SELECT id, request_id, recommendation, risk_score, confidence, reason, created_at
FROM ai_decisions
WHERE request_id = $1
ORDER BY created_at DESC;
