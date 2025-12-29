package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/models"
	apperrors "github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/errors"
	"github.com/labstack/echo/v4"
)

const (
	MsgOrderRetrieved = "Order retrieved successfully"
	MsgOrderCreated   = "Order created successfully"
	MsgOrderUpdated   = "Order updated successfully"
	MsgOrderCanceled  = "Order canceled successfully"

	MsgFailedToRetrieveOrder = "Failed to retrieve order"
	MsgFailedToCreateOrder   = "Failed to create order"
	MsgFailedToUpdateOrder   = "Failed to update order"
	MsgFailedToDeleteOrder   = "Failed to delete order"
)

func respondSuccess(c echo.Context, status int, message string, data interface{}) error {
	return c.JSON(status, models.SuccessResponse{
		Message: message,
		Data:    data,
	})
}

func respondError(c echo.Context, status int, err error) error {
	return c.JSON(status, models.ErrorResponse{
		Error: err.Error(),
	})
}

func handleGetError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrInvalidUserInput),
		errors.Is(err, apperrors.ErrInvalidCartOperation):
		return respondError(c, http.StatusBadRequest, err)

	case errors.Is(err, apperrors.ErrInsufficientStock),
		errors.Is(err, apperrors.ErrCartAlreadyCheckedOut):
		return respondError(c, http.StatusForbidden, err)

	case errors.Is(err, apperrors.ErrInternalServerError):
		return respondError(c, http.StatusInternalServerError, err)

	default:
		log.Printf("Unhandled inventory error: %v", err)
		return respondError(c, http.StatusInternalServerError, fmt.Errorf("%w: %s", apperrors.ErrInternalServerError, err))
	}
}

func handleOperationError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, apperrors.ErrProductNotBelongToSeller),
		errors.Is(err, apperrors.ErrInvalidUserInput),
		errors.Is(err, apperrors.ErrInvalidCartOperation):
		return respondError(c, http.StatusForbidden, err)

	case err.Error() == apperrors.ErrInvalidProductUpdatePayload.Error(),
		errors.Is(err, apperrors.ErrInsufficientStock),
		errors.Is(err, apperrors.ErrCartAlreadyCheckedOut):
		return respondError(c, http.StatusBadRequest, apperrors.ErrInvalidRequestPayload)

	default:
		c.Logger().Errorf("%s: %v", apperrors.ErrInternalServerError, err)
		return respondError(c, http.StatusInternalServerError, fmt.Errorf("internal server error"))
	}
}
