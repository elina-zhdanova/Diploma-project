package workflow

import (
	"github.com/itshop/api/internal/pgxutil"
	"github.com/jackc/pgx/v5/pgtype"
)

// Цепочка согласующих (как в демо Gatekeeper: бизнес → ИБ → IT).
// UUID совпадают с seed `000003_seed_demo.up.sql`.
func DefaultApproverChain() []pgtype.UUID {
	return []pgtype.UUID{
		pgxutil.MustParse("c0000002-0000-0000-0000-000000000002"), // approver
		pgxutil.MustParse("c0000003-0000-0000-0000-000000000003"), // security
		pgxutil.MustParse("c0000004-0000-0000-0000-000000000004"), // admin
	}
}
