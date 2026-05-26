package workflow

import (
	"testing"

	"github.com/itshop/api/internal/pgxutil"
	"github.com/jackc/pgx/v5/pgtype"
)

func u(s string) pgtype.UUID {
	return pgxutil.MustParse(s)
}

func TestMergeApproverChains_overlapAndNoDupAcrossChains(t *testing.T) {
	A := u("c0000002-0000-0000-0000-000000000002")
	B := u("c0000003-0000-0000-0000-000000000003")
	C := u("c0000004-0000-0000-0000-000000000004")

	got := MergeApproverChains([][]pgtype.UUID{
		{A, B},
		{B, C},
	})
	if len(got) != 3 || !sameApproverID(got[0], A) || !sameApproverID(got[1], B) || !sameApproverID(got[2], C) {
		t.Fatalf("want [A,B,C], got %v", got)
	}

	got2 := MergeApproverChains([][]pgtype.UUID{
		{A, B},
		{A, B},
	})
	if len(got2) != 2 || !sameApproverID(got2[0], A) || !sameApproverID(got2[1], B) {
		t.Fatalf("identical chains must collapse to [A,B], got len=%d", len(got2))
	}
}
