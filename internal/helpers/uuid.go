package helpers

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/errors"
)

/*
GenerateNewUserID digunakan agar skalabilitas, desentralisasi, keamanan, dan fleksibilitas terjamin

	skalabilitas -> Menghindari bottleneck saat data tubuh besar
	desentralisasi -> mengurangi otonomi dan meningkatkan coupling antar komponen
	global uniquesness without coordination -> tidak khawatir akan tabrakan ID
	fleksibilitas -> memberikan ID yang dapat dibagikan dan dijamin unik di seluruh ekosistem, tanpa perlu khawatir tentang konflik ID dengan sistem eksternal
*/
func GenerateNewID() uuid.UUID {
	return uuid.New()
}

func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func GetIDFromPathParam(c echo.Context, key string) (uuid.UUID, error) {
	val := c.Param(key)
	if val == "" || !IsValidUUID(val) {
		return uuid.Nil, errors.ErrInvalidRequestPayload
	}

	res, err := StringToUUID(val)
	if err != nil {
		return uuid.Nil, err
	}

	return res, nil
}

func GetFromPathParam(c echo.Context, key string) (string, error) {
	val := c.Param(key)
	if val == "" {
		return "", errors.ErrInvalidRequestPayload
	}

	return val, nil
}
