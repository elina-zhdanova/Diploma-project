package pgxutil

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func UUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

func ToGoogle(u pgtype.UUID) (uuid.UUID, bool) {
	if !u.Valid {
		return uuid.Nil, false
	}
	return uuid.UUID(u.Bytes), true
}

func ParseUUID(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return UUID(u), nil
}

func MustParse(s string) pgtype.UUID {
	u, err := ParseUUID(s)
	if err != nil {
		panic(err)
	}
	return u
}
