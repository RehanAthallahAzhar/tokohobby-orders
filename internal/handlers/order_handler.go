package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/models"
	apperrors "github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/errors"
)

func (api *API) CreateOrder() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		api.log.Infof("Received request to create product from IP: %s", c.RealIP())

		userID, err := getUserIDFromContext(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, apperrors.ErrInvalidUserSession)
		}

		var req models.OrderDetailReq
		if err := c.Bind(&req); err != nil {
			return respondError(c, http.StatusBadRequest, apperrors.ErrInvalidRequestPayload)
		}
		api.log.WithField("user_id", userID).Info("Receiving CreateOrder requests")

		order, err := api.OrderSvc.CreateOrder(ctx, userID, req)
		if err != nil {
			return handleOperationError(c, err)
		}

		return respondSuccess(c, http.StatusCreated, MsgOrderCreated, toOrderResponse(order))
	}
}

func (api *API) GetUserOrders() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		userID, err := getUserIDFromContext(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, apperrors.ErrInvalidUserSession)
		}

		api.log.WithField("user_id", userID).Info("Receiving GetUserOrders requests")

		orderDetails, err := api.OrderSvc.GetOrdersByUserID(ctx, userID)
		if err != nil {
			api.log.WithField("error", err).Error("Error from the service when retrieving orders")
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusCreated, MsgOrderRetrieved, toOrderDetailsListResponse(orderDetails))

	}
}

func (api *API) GetOrderDetails() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		orderID, err := getIDFromPathParam(c, "id")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		api.log.WithField("order_id", orderID).Info("Receiving GetOrderItems requests")

		items, err := api.OrderSvc.GetOrderItemsByOrderID(ctx, orderID)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgOrderRetrieved, toOrderItemResponse(items))
	}
}

func (api *API) CancelOrder() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		orderID, err := getIDFromPathParam(c, "id")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		userID, err := getUserIDFromContext(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, apperrors.ErrInvalidUserSession)
		}

		api.log.WithField("order_id", orderID).Info("Receiving CancelOrder requests")

		order, err := api.OrderSvc.CancelOrder(ctx, orderID, userID)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgOrderCanceled, toOrderResponse(order))
	}
}

func (api *API) ResetAllOrderCaches() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		err := api.OrderSvc.ResetAllOrderCaches(ctx)
		if err != nil {
			return handleOperationError(c, err)
		}
		return nil
	}

}

// ------- HELPERS -------
func toOrderDetailsListResponse(details []entities.OrderDetails) []models.OrderDetailsResponse {
	respList := make([]models.OrderDetailsResponse, len(details))
	for i, d := range details {
		respList[i] = models.OrderDetailsResponse{
			Order: *toOrderResponse(&d.Order),
			Items: toOrderItemResponse(d.Items),
		}
	}
	return respList
}

func toOrderResponse(order *entities.Order) *models.OrderResponse {
	return &models.OrderResponse{
		ID:              order.ID.String(),
		UserID:          order.UserID.String(),
		Status:          order.OrderStatus,
		TotalPrice:      order.TotalPrice,
		ShippingAddress: order.ShippingAddress,
		CreatedAt:       order.CreatedAt,
	}
}

func toOrderItemResponse(items []entities.OrderItem) []models.OrderItemRes {
	respList := make([]models.OrderItemRes, 0, len(items))
	for _, item := range items {
		respList = append(respList, models.OrderItemRes{
			ID:              item.ID,
			OrderID:         item.OrderID,
			SellerID:        item.SellerID,
			SellerName:      item.SellerName,
			ProductID:       item.ProductID,
			ProductName:     item.ProductName,
			Quantity:        item.Quantity,
			ProductPrice:    item.ProductPrice,
			CartDescription: item.CartDescription,
		})
	}
	return respList
}
