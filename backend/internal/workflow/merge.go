package workflow

import (
	"github.com/jackc/pgx/v5/pgtype"
)

func sameApproverID(a, b pgtype.UUID) bool {
	if !a.Valid || !b.Valid {
		return false
	}
	return a.Bytes == b.Bytes
}

// suffixPrefixOverlapLen — максимальное k, при котором суффикс a длиной k совпадает с префиксом b
// (поэлементно через sameApproverID). Нужно, чтобы склеивать цепочки разных ролей без дублирования стыка.
func suffixPrefixOverlapLen(a, b []pgtype.UUID) int {
	max := len(a)
	if len(b) < max {
		max = len(b)
	}
	for k := max; k >= 1; k-- {
		ok := true
		for i := 0; i < k; i++ {
			if !sameApproverID(a[len(a)-k+i], b[i]) {
				ok = false
				break
			}
		}
		if ok {
			return k
		}
	}
	return 0
}

// MergeApproverChains объединяет цепочки по порядку уникальных ролей в заявке.
// Стык между цепочками: максимальное перекрытие «хвост/голова» убирается (как в [A,B] + [B,C] → [A,B,C]).
// Подряд идущие одинаковые согласующие внутри результата схлопываются.
// Раньше [A,B]+[A,B] ошибочно превращалось в [A,B,A,B].
func MergeApproverChains(chains [][]pgtype.UUID) []pgtype.UUID {
	if len(chains) == 0 {
		return nil
	}
	out := append([]pgtype.UUID(nil), chains[0]...)
	for ci := 1; ci < len(chains); ci++ {
		next := chains[ci]
		if len(out) == 0 {
			out = append(out, next...)
			continue
		}
		if len(next) == 0 {
			continue
		}
		o := suffixPrefixOverlapLen(out, next)
		for _, u := range next[o:] {
			if len(out) == 0 || !sameApproverID(out[len(out)-1], u) {
				out = append(out, u)
			}
		}
	}
	return out
}
