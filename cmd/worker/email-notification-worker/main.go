package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	rabbitmq "github.com/RehanAthallahAzhar/tokohobby-messaging/rabbitmq"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/configs"
	orderMsg "github.com/RehanAthallahAzhar/tokohobby-orders/internal/messaging"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/logger"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logger.NewLogger()
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.DebugLevel)

	cfg, err := configs.LoadConfig(log)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Info("🚀 Starting Order Email Notification Worker...")

	// Initialize RabbitMQ
	rmqConfig := &rabbitmq.RabbitMQConfig{
		URL:            cfg.RabbitMQ.URL,
		MaxRetries:     cfg.RabbitMQ.MaxRetries,
		RetryDelay:     cfg.RabbitMQ.RetryDelay,
		PrefetchCount:  cfg.RabbitMQ.PrefetchCount,
		ReconnectDelay: cfg.RabbitMQ.ReconnectDelay,
	}
	rmq, err := rabbitmq.NewRabbitMQ(rmqConfig)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rmq.Close()

	log.Info("Connected to RabbitMQ successfully")

	// Setup queue
	if err := rabbitmq.SetupOrderEmailQueue(rmq); err != nil {
		log.Fatalf("Failed to setup queue: %v", err)
	}

	emailService := NewEmailService()

	// Message handler
	handler := func(ctx context.Context, body []byte) error {
		var event orderMsg.OrderStatusChangedEvent

		if err := rabbitmq.UnmarshalMessage(body, &event); err != nil {
			return fmt.Errorf("failed to unmarshal: %w", err)
		}

		log.Printf("📧 Processing email notification for order %s: %s → %s",
			event.OrderID, event.OldStatus, event.NewStatus)

		// Send email based on status
		return emailService.SendOrderStatusEmail(event)
	}

	// Create consumer - consume ALL status changes
	consumerOpts := rabbitmq.ConsumerOptions{
		QueueName:   "order.email.notifications",
		WorkerCount: 5, // 5 concurrent workers
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

type EmailService struct{}

func NewEmailService() *EmailService {
	return &EmailService{}
}

func (s *EmailService) SendOrderStatusEmail(event orderMsg.OrderStatusChangedEvent) error {
	// Different email template based on status
	templates := map[orderMsg.OrderStatus]string{
		orderMsg.OrderStatusPaid:      "✅ Payment received!",
		orderMsg.OrderStatusShipped:   "📦 Your order has been shipped!",
		orderMsg.OrderStatusDelivered: "🎉 Order delivered successfully!",
		orderMsg.OrderStatusCancelled: "❌ Order cancelled",
	}

	template, exists := templates[event.NewStatus]
	if !exists {
		template = "Order status updated"
	}

	log.Printf("✅ [MOCK] Email sent to %s: %s (Order: %s)",
		event.Email, template, event.OrderID)
	return nil
}
