package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	rabbitmq "github.com/RehanAthallahAzhar/tokohobby-messaging/rabbitmq"
	orderMsg "github.com/RehanAthallahAzhar/tokohobby-orders/internal/messaging"
)

func main() {
	log.Println("🚀 Starting Order History Worker...")

	// Initialize RabbitMQ
	rmqConfig := rabbitmq.DefaultConfig()
	rmq, err := rabbitmq.NewRabbitMQ(rmqConfig)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rmq.Close()

	// Setup queue - selective binding (paid & delivered only)
	if err := rabbitmq.SetupOrderHistoryQueue(rmq); err != nil {
		log.Fatalf("Failed to setup queue: %v", err)
	}

	historyService := NewHistoryService()

	// Message handler
	handler := func(ctx context.Context, body []byte) error {
		var event orderMsg.OrderStatusChangedEvent

		if err := rabbitmq.UnmarshalMessage(body, &event); err != nil {
			return fmt.Errorf("failed to unmarshal: %w", err)
		}

		log.Printf("📝 Updating user history for order %s: %s",
			event.OrderID, event.NewStatus)

		// Only process paid & delivered
		return historyService.UpdateUserOrderHistory(event)
	}

	// Create consumer
	consumerOpts := rabbitmq.ConsumerOptions{
		QueueName:   "order.user.history",
		WorkerCount: 3,
		AutoAck:     false,
	}
	consumer := rabbitmq.NewConsumer(rmq, consumerOpts, handler)

	// Start consuming
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := consumer.Start(ctx); err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down worker...")
	cancel()
}

type HistoryService struct{}

func NewHistoryService() *HistoryService {
	return &HistoryService{}
}

func (s *HistoryService) UpdateUserOrderHistory(event orderMsg.OrderStatusChangedEvent) error {
	// TODO: Update database - user order history
	log.Printf("✅ [MOCK] User history updated: User %s, Order %s - %s",
		event.UserID, event.OrderID, event.NewStatus)
	return nil
}
