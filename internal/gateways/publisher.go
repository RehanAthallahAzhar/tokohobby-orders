package gateway

import (
	"context"

	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/models"
)

type EventPublisher interface {
	PublishOrderCreated(ctx context.Context, event models.OrderCreatedEvent) error
	// PublishOrderCanceled (coming soon)
}
