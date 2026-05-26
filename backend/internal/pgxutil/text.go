package pgxutil

import "github.com/jackc/pgx/v5/pgtype"

func Text(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func TextOptional(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return Text(s)
}
