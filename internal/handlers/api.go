package handlers

import (
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/helpers"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/services"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type API struct {
	OrderSvc services.OrderService
	log      *logrus.Logger
}

func NewHandler(
	orderSvc services.OrderService,
	log *logrus.Logger,
) *API {
	return &API{
		OrderSvc: orderSvc,
		log:      log,
	}
}

// ---- HELPERS -----

func getUserIDFromContext(c echo.Context) (uuid.UUID, error) {
	if val := c.Get("userID"); val != nil {
		if id, ok := val.(string); ok {

			res, err := helpers.StringToUUID(id)
			if err != nil {
				return uuid.Nil, err
			}

			return res, nil

		}
	}

	return uuid.Nil, errors.ErrInvalidUserSession
}

func getIDFromPathParam(c echo.Context, key string) (uuid.UUID, error) {
	val := c.Param(key)
	if val == "" || !helpers.IsValidUUID(val) {
		return uuid.Nil, errors.ErrInvalidRequestPayload
	}

	res, err := helpers.StringToUUID(val)
	if err != nil {
		return uuid.Nil, err
	}

	return res, nil
}

// func getFromPathParam(c echo.Context, key string) (string, error) {
// 	val := c.Param(key)
// 	if val == "" {
// 		return "", errors.ErrInvalidRequestPayload
// 	}

// 	return val, nil
// }
