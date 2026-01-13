package messaging

import (
	"context"
	"fmt"

	messaging "github.com/RehanAthallahAzhar/tokohobby-messaging-go"
	"github.com/sirupsen/logrus"
)

type EventPublisher struct {
	publisher *messaging.Publisher
	log       *logrus.Logger
}

func NewEventPublisher(rmq *messaging.RabbitMQ, log *logrus.Logger) *EventPublisher {
	return &EventPublisher{
		publisher: messaging.NewPublisher(rmq),
		log:       log,
	}
}

// PublishOrderStatusChanged publish event order status changed
func (p *EventPublisher) PublishOrderStatusChanged(ctx context.Context, event OrderStatusChangedEvent) error {
	// Add type field to event
	eventWithType := struct {
		Type string `json:"type"`
		OrderStatusChangedEvent
	}{
		Type:                    "order.status.changed",
		OrderStatusChangedEvent: event,
	}

	// Dynamic routing key based on status
	routingKey := fmt.Sprintf("order.status.%s", event.NewStatus)

	opts := messaging.PublishOptions{
		Exchange:   "order.events",
		RoutingKey: routingKey, // order.status.paid, order.status.shipped, etc
		Mandatory:  false,
		Immediate:  false,
	}

	err := p.publisher.Publish(ctx, opts, eventWithType)
	if err != nil {
		p.log.Errorf("Failed to publish order status changed event: %v", err)
		return err
	}

	p.log.Debugf("Published order.status.%s event for order: %s", event.NewStatus, event.OrderID)
	return nil
}

// PublishOrderCreated publish event order created
func (p *EventPublisher) PublishOrderCreated(ctx context.Context, event OrderCreatedEvent) error {
	eventWithType := struct {
		Type string `json:"type"`
		OrderCreatedEvent
	}{
		Type:              "order.created",
		OrderCreatedEvent: event,
	}

	opts := messaging.PublishOptions{
		Exchange:   "order.events",
		RoutingKey: "order.created",
		Mandatory:  false,
		Immediate:  false,
	}

	err := p.publisher.Publish(ctx, opts, eventWithType)
	if err != nil {
		p.log.Errorf("Failed to publish order created event: %v", err)
		return err
	}

	p.log.Infof("Published order.created event for order: %s", event.OrderID)
	return nil
}

// PublishOrderShipped publish event order shipped
func (p *EventPublisher) PublishOrderShipped(ctx context.Context, event OrderShippedEvent) error {
	eventWithType := struct {
		Type string `json:"type"`
		OrderShippedEvent
	}{
		Type:              "order.shipped",
		OrderShippedEvent: event,
	}

	opts := messaging.PublishOptions{
		Exchange:   "order.events",
		RoutingKey: "order.shipped",
		Mandatory:  false,
		Immediate:  false,
	}

	err := p.publisher.Publish(ctx, opts, eventWithType)
	if err != nil {
		p.log.Errorf("Failed to publish order shipped event: %v", err)
		return err
	}

	p.log.Infof("Published order.shipped event for order: %s", event.OrderID)
	return nil
}
