package routes

import (
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/handlers"

	"github.com/labstack/echo/v4"
)

func InitRoutes(e *echo.Echo, orderHandler *handlers.OrderHandler, authMiddleware echo.MiddlewareFunc) {
	order := e.Group("/api/orders")
	order.Use(authMiddleware)
	{
		order.POST("/", orderHandler.CreateOrder())
		order.GET("/", orderHandler.GetUserOrders())
		order.GET("/:id", orderHandler.GetOrderDetails())
		order.POST("/:id/cancel", orderHandler.CancelOrder())
		order.POST("/reset-caches", orderHandler.ResetAllOrderCaches())
	}
}
