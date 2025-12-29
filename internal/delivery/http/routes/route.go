package routes

import (
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/handlers"

	"github.com/labstack/echo/v4"
)

func InitRoutes(e *echo.Echo, handler *handlers.API, authMiddleware echo.MiddlewareFunc) {
	apiV1 := e.Group("/api/v1")

	apiV1.Use(authMiddleware)

	orderGroup := apiV1.Group("/orders")
	{
		orderGroup.POST("/", handler.CreateOrder())
		orderGroup.GET("/", handler.GetUserOrders())
		orderGroup.GET("/:id", handler.GetOrderDetails())
		orderGroup.POST("/:id/cancel", handler.CancelOrder())
		orderGroup.POST("/reset-caches", handler.ResetAllOrderCaches())
	}
}
