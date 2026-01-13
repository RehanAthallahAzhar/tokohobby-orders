package helpers

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/google/uuid"
)

func StringToUUID(id string) (uuid.UUID, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid UUID: %w", err)
	}
	return uuidID, nil
}

func IntToNullInt32(val int) sql.NullInt32 {
	return sql.NullInt32{
		Int32: int32(val),
		Valid: true,
	}
}

func StringToNullString(val string) sql.NullString {
	return sql.NullString{
		String: val,
		Valid:  true,
	}
}

func StringToFloat64(val string) (float64, error) {
	return strconv.ParseFloat(val, 64)
}
