package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/itshop/api/internal/pgxutil"
	"github.com/itshop/api/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
)

// AllRequestItemsAreLowRisk — все позиции заявки в каталоге с уровнем риска low (без medium/high).
func AllRequestItemsAreLowRisk(ctx context.Context, q *store.Queries, requestID pgtype.UUID) (bool, error) {
	items, err := q.ListRequestItems(ctx, requestID)
	if err != nil {
		return false, err
	}
	if len(items) == 0 {
		return false, nil
	}
	for _, it := range items {
		if !strings.EqualFold(strings.TrimSpace(it.RiskLevel), "low") {
			return false, nil
		}
	}
	return true, nil
}

// AnalyzeRequest — rule-based MVP: оценка по risk_level позиций заявки и запись в ai_decisions.
func AnalyzeRequest(ctx context.Context, q *store.Queries, requestID pgtype.UUID) (store.AiDecision, error) {
	items, err := q.ListRequestItems(ctx, requestID)
	if err != nil {
		return store.AiDecision{}, err
	}
	maxScore := 0.0
	hasHigh := false
	for _, it := range items {
		switch it.RiskLevel {
		case "high":
			hasHigh = true
			maxScore = max(maxScore, 85)
		case "medium":
			maxScore = max(maxScore, 55)
		default:
			maxScore = max(maxScore, 25)
		}
	}
	if len(items) == 0 {
		maxScore = 10
	}

	rec := "approve"
	conf := 0.82
	reason := "Низкий совокупный риск по справочнику access_roles."
	if hasHigh {
		rec = "manual_review"
		conf = 0.71
		reason = "Обнаружены позиции с высоким уровнем риска — требуется ручная проверка."
	} else if maxScore >= 50 {
		rec = "approve_with_monitoring"
		conf = 0.76
		reason = fmt.Sprintf("Средний совокупный балл риска: %.0f.", maxScore)
	}

	ad, err := q.InsertAIDecision(ctx, store.InsertAIDecisionParams{
		RequestID:      requestID,
		Recommendation: rec,
		RiskScore:      maxScore,
		Confidence:     conf,
		Reason:         pgxutil.Text(reason),
	})
	if err != nil {
		return store.AiDecision{}, err
	}
	if err := q.UpdateRequestRisk(ctx, store.UpdateRequestRiskParams{
		ID: requestID,
		RiskScore: pgtype.Float8{
			Float64: maxScore,
			Valid:   true,
		},
	}); err != nil {
		return store.AiDecision{}, err
	}
	return ad, nil
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
