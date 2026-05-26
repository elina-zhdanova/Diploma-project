package workflow

import (
	"context"

	"github.com/itshop/api/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
)

// ApproverChainForAccessRoles строит цепочку согласований для заявки по наборам ролей доступа (порядок — первое вхождение в заявке).
// Если у какой-либо роли нет настроенных согласующих, используется DefaultApproverChain.
func ApproverChainForAccessRoles(ctx context.Context, q *store.Queries, roleIDs []pgtype.UUID) ([]pgtype.UUID, error) {
	if len(roleIDs) == 0 {
		return DefaultApproverChain(), nil
	}
	chains := make([][]pgtype.UUID, 0, len(roleIDs))
	for _, rid := range roleIDs {
		ids, err := q.ListApproversOrderedForAccessRole(ctx, rid)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return DefaultApproverChain(), nil
		}
		chains = append(chains, ids)
	}
	merged := MergeApproverChains(chains)
	if len(merged) == 0 {
		return DefaultApproverChain(), nil
	}
	return merged, nil
}
