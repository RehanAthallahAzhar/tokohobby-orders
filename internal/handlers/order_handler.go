package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/messaging"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/models"
	apperrors "github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/services"
)

type OrderHandler struct {
	OrderSvc services.OrderService
	log      *logrus.Logger
}

func NewOrderHandler(
	orderSvc services.OrderService,
	log *logrus.Logger,
) *OrderHandler {
	return &OrderHandler{
		OrderSvc: orderSvc,
		log:      log,
	}
}

func (h *OrderHandler) CreateOrder() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		h.log.Infof("Received request to create product from IP: %s", c.RealIP())

		userID, err := getUserIDFromContext(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, apperrors.ErrInvalidUserSession)
		}

		var req models.OrderDetailReq
		if err := c.Bind(&req); err != nil {
			return respondError(c, http.StatusBadRequest, apperrors.ErrInvalidRequestPayload)
		}

		order, err := h.OrderSvc.CreateOrder(ctx, userID, req)
		if err != nil {
			return handleOperationError(c, err)
		}

		return respondSuccess(c, http.StatusCreated, MsgOrderCreated, toOrderResponse(order))
	}
}

func (h *OrderHandler) GetUserOrders() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		userID, err := getUserIDFromContext(c)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, apperrors.ErrInvalidUserSession)
		}

		orderDetails, err := h.OrderSvc.GetOrdersByUserID(ctx, userID)
		if err != nil {
			h.log.WithField("error", err).Error("Error from the service when retrieving orders")
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusCreated, MsgOrderRetrieved, toOrderDetailsListResponse(orderDetails))

	}
}

func (h *OrderHandler) GetOrderDetails() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		orderID, err := getIDFromPathParam(c, "id")
		if err != nil {
			return respondError(c, http.StatusBadRequest, err)
		}

		h.log.WithField("order_id", orderID).Info("Receiving GetOrderItems requests")

		items, err := h.OrderSvc.GetOrderItemsByOrderID(ctx, orderID)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgOrderRetrieved, toOrderItemResponse(items))
	}
}

func (h *OrderHandler) UpdateOrderStatus(c echo.Context) error {
	ctx := c.Request().Context()

	orderID, err := getIDFromPathParam(c, "id")
	if err != nil {
		return respondError(c, http.StatusBadRequest, err)
	}

	var req models.UpdateOrderStatusReq
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, apperrors.ErrInvalidRequestPayload)
	}

	var order *entities.Order
	order, err = h.OrderSvc.UpdateOrderStatus(ctx, orderID, messaging.OrderStatus(req.Status))
	if err != nil {
		return handleGetError(c, err)
	}

	return respondSuccess(c, http.StatusOK, "Order status updated", order)
}

func (h *OrderHandler) CancelOrder() echo.HandlerFunc {
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

		h.log.WithField("order_id", orderID).Info("Receiving CancelOrder requests")

		order, err := h.OrderSvc.CancelOrder(ctx, orderID, userID)
		if err != nil {
			return handleGetError(c, err)
		}

		return respondSuccess(c, http.StatusOK, MsgOrderCanceled, toOrderResponse(order))
	}
}

func (h *OrderHandler) ResetAllOrderCaches() echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		err := h.OrderSvc.ResetAllOrderCaches(ctx)
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
