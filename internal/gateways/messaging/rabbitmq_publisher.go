package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/models"
	"github.com/streadway/amqp"
)

const (
	OrderExchange = "order_events"
)

type RabbitMQPublisher struct {
	channel *amqp.Channel
}

func NewRabbitMQPublisher(ch *amqp.Channel) (*RabbitMQPublisher, error) {

	err := ch.ExchangeDeclare(
		OrderExchange, // name
		"fanout",      // type
		true,          // durable
		false,         // auto-deleted
		false,         // internal
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mendeklarasikan exchange: %w", err)
	}

	return &RabbitMQPublisher{channel: ch}, nil
}

func (p *RabbitMQPublisher) PublishOrderCreated(ctx context.Context, event models.OrderCreatedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = p.channel.Publish(
		OrderExchange, // exchange
		"",            // routing key
		false,         // mandatory
		false,         // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	log.Printf("OrderCreated event was successfully published for Order ID: %s", event.OrderID)
	return nil
}
