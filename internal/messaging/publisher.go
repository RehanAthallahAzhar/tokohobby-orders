package messaging

import (
	"context"
	"fmt"

	rabbitmq "github.com/RehanAthallahAzhar/tokohobby-messaging/rabbitmq"
	"github.com/sirupsen/logrus"
)

type EventPublisher interface {
	PublishOrderStatusChanged(ctx context.Context, event OrderStatusChangedEvent) error
	PublishOrderCreated(ctx context.Context, event OrderCreatedEvent) error
	PublishOrderShipped(ctx context.Context, event OrderShippedEvent) error
	PublishOrderPaid(ctx context.Context, event OrderPaidEvent) error
	PublishOrderDelivered(ctx context.Context, event OrderDeliveredEvent) error
	PublishOrderCancelled(ctx context.Context, event OrderCancelledEvent) error
	PublishOrderRefunded(ctx context.Context, event OrderRefundedEvent) error
}

type EventPublisherImpl struct {
	rabbitmq *rabbitmq.Publisher
	log      *logrus.Logger
}

func NewEventPublisher(rmq *rabbitmq.RabbitMQ, log *logrus.Logger) *EventPublisherImpl {
	return &EventPublisherImpl{
		rabbitmq: rabbitmq.NewPublisher(rmq),
		log:      log,
	}
}

// PublishOrderStatusChanged publish event order status changed
func (p *EventPublisherImpl) PublishOrderStatusChanged(ctx context.Context, event OrderStatusChangedEvent) error {
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

	opts := rabbitmq.PublishOptions{
		Exchange:   "order.events",
		RoutingKey: routingKey, // order.status.paid, order.status.shipped, etc
		Mandatory:  false,
		Immediate:  false,
	}

	err := p.rabbitmq.Publish(ctx, opts, eventWithType)
	if err != nil {
		p.log.Errorf("Failed to publish order status changed event: %v", err)
		return err
	}

	p.log.Debugf("Published order.status.%s event for order: %s", event.NewStatus, event.OrderID)
	return nil
}

// PublishOrderCreated publish event order created
func (p *EventPublisherImpl) PublishOrderCreated(ctx context.Context, event OrderCreatedEvent) error {
	eventWithType := struct {
		Type string `json:"type"`
		OrderCreatedEvent
	}{
		Type:              "order.created",
		OrderCreatedEvent: event,
	}

	opts := rabbitmq.PublishOptions{
		Exchange:   "order.events",
		RoutingKey: "order.created",
		Mandatory:  false,
		Immediate:  false,
	}

	err := p.rabbitmq.Publish(ctx, opts, eventWithType)
	if err != nil {
		p.log.Errorf("Failed to publish order created event: %v", err)
		return err
	}

	p.log.Infof("Published order.created event for order: %s", event.OrderID)
	return nil
}

// PublishOrderShipped publish event order shipped
func (p *EventPublisherImpl) PublishOrderShipped(ctx context.Context, event OrderShippedEvent) error {
	eventWithType := struct {
		Type string `json:"type"`
		OrderShippedEvent
	}{
		Type:              "order.shipped",
		OrderShippedEvent: event,
	}

	opts := rabbitmq.PublishOptions{
		Exchange:   "order.events",
		RoutingKey: "order.shipped",
		Mandatory:  false,
		Immediate:  false,
	}

	err := p.rabbitmq.Publish(ctx, opts, eventWithType)
	if err != nil {
		p.log.Errorf("Failed to publish order shipped event: %v", err)
		return err
	}

	p.log.Infof("Published order.shipped event for order: %s", event.OrderID)
	return nil
}

// PublishOrderPaid publish event order paid
func (p *EventPublisherImpl) PublishOrderPaid(ctx context.Context, event OrderPaidEvent) error {
	eventWithType := struct {
		Type string `json:"type"`
		OrderPaidEvent
	}{
		Type:           "order.paid",
		OrderPaidEvent: event,
	}

	opts := rabbitmq.PublishOptions{
		Exchange:   "order.events",
		RoutingKey: "order.paid",
		Mandatory:  false,
		Immediate:  false,
	}

	err := p.rabbitmq.Publish(ctx, opts, eventWithType)
	if err != nil {
		p.log.Errorf("Failed to publish order paid event: %v", err)
		return err
	}

	p.log.Infof("Published order.paid event for order: %s", event.OrderID)
	return nil
}

// PublishOrderDelivered publish event order delivered
func (p *EventPublisherImpl) PublishOrderDelivered(ctx context.Context, event OrderDeliveredEvent) error {
	eventWithType := struct {
		Type string `json:"type"`
		OrderDeliveredEvent
	}{
		Type:                "order.delivered",
		OrderDeliveredEvent: event,
	}

	opts := rabbitmq.PublishOptions{
		Exchange:   "order.events",
		RoutingKey: "order.delivered",
		Mandatory:  false,
		Immediate:  false,
	}

	err := p.rabbitmq.Publish(ctx, opts, eventWithType)
	if err != nil {
		p.log.Errorf("Failed to publish order delivered event: %v", err)
		return err
	}

	p.log.Infof("Published order.delivered event for order: %s", event.OrderID)
	return nil
}

// PublishOrderCancelled publish event order cancelled
func (p *EventPublisherImpl) PublishOrderCancelled(ctx context.Context, event OrderCancelledEvent) error {
	eventWithType := struct {
		Type string `json:"type"`
		OrderCancelledEvent
	}{
		Type:                "order.cancelled",
		OrderCancelledEvent: event,
	}

	opts := rabbitmq.PublishOptions{
		Exchange:   "order.events",
		RoutingKey: "order.cancelled",
		Mandatory:  false,
		Immediate:  false,
	}

	err := p.rabbitmq.Publish(ctx, opts, eventWithType)
	if err != nil {
		p.log.Errorf("Failed to publish order cancelled event: %v", err)
		return err
	}

	p.log.Infof("Published order.cancelled event for order: %s", event.OrderID)
	return nil
}

// PublishOrderRefunded publish event order refunded
func (p *EventPublisherImpl) PublishOrderRefunded(ctx context.Context, event OrderRefundedEvent) error {
	eventWithType := struct {
		Type string `json:"type"`
		OrderRefundedEvent
	}{
		Type:               "order.refunded",
		OrderRefundedEvent: event,
	}

	opts := rabbitmq.PublishOptions{
		Exchange:   "order.events",
		RoutingKey: "order.refunded",
		Mandatory:  false,
		Immediate:  false,
	}

	err := p.rabbitmq.Publish(ctx, opts, eventWithType)
	if err != nil {
		p.log.Errorf("Failed to publish order refunded event: %v", err)
		return err
	}

	p.log.Infof("Published order.refunded event for order: %s", event.OrderID)
	return nil
}
