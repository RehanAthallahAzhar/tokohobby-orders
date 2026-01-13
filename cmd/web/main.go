package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/RehanAthallahAzhar/tokohobby-orders/db"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/configs"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/handlers"
	dbGenerated "github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/db"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	customMiddleware "github.com/RehanAthallahAzhar/tokohobby-orders/internal/delivery/http/middlewares"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/delivery/http/routes"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/grpc/account"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/logger"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/redis"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/repositories"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/services"

	accountpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/account"
	authpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/auth"
	productpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/product"

	messaging "github.com/RehanAthallahAzhar/tokohobby-messaging-go"
	orderMsg "github.com/RehanAthallahAzhar/tokohobby-orders/internal/messaging"
)

func main() {
	log := logger.NewLogger()

	cfg, err := configs.LoadConfig(log)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dbCredential := models.Credential{
		Host:         cfg.Postgre.Host,
		Username:     cfg.Postgre.User,
		Password:     cfg.Postgre.Password,
		DatabaseName: cfg.Postgre.Name,
		Port:         cfg.Postgre.Port,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := db.Connect(ctx, &dbCredential)
	if err != nil {
		log.Errorf("DB connection error: %v", err)
	}

	// Migration
	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		dbCredential.Username,
		dbCredential.Password,
		dbCredential.Host,
		dbCredential.Port,
		dbCredential.DatabaseName,
	)
	m, err := migrate.New(
		cfg.Migration.Path,
		connectionString,
	)
	if err != nil {
		log.Fatalf("Failed to create migration instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to execute database migrations: %v", err)
	}

	// Init SQLC Store for transaction
	sqlcQueries := dbGenerated.New(conn)
	store := dbGenerated.NewStore(conn)

	// Redis
	redisClient, err := redis.NewRedisClient(&cfg.Redis, log)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer redisClient.Close()

	// gRPC Product
	productConn := createGrpcConnection(cfg.GRPC.ProductServiceAddress, log)
	defer productConn.Close()

	productClient := productpb.NewProductServiceClient(productConn)

	// gRPC Account & Auth
	accountConn := createGrpcConnection(cfg.GRPC.AccountServiceAddress, log)
	defer accountConn.Close()
	accountClient := accountpb.NewAccountServiceClient(accountConn)

	authClient := authpb.NewAuthServiceClient(accountConn)
	authClientWrapper := account.NewAuthClientFromService(authClient, accountConn)

	// Initialize RabbitMQ
	rmqConfig := &messaging.RabbitMQConfig{
		URL:            cfg.RabbitMQ.URL,
		MaxRetries:     cfg.RabbitMQ.MaxRetries,
		RetryDelay:     cfg.RabbitMQ.RetryDelay,
		PrefetchCount:  cfg.RabbitMQ.PrefetchCount,
		ReconnectDelay: cfg.RabbitMQ.ReconnectDelay,
	}
	rmq, err := messaging.NewRabbitMQ(rmqConfig)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rmq.Close()

	// Setup exchange
	if err := messaging.SetupOrderExchange(rmq); err != nil {
		log.Fatalf("Failed to setup order exchange: %v", err)
	}

	// Create event publisher
	eventPublisher := orderMsg.NewEventPublisher(rmq, log)

	// Dependency Injection
	validate := validator.New()
	orderRepo := repositories.NewOrderRepository(conn, sqlcQueries, store, log)
	orderService := services.NewOrderService(orderRepo, redisClient, productClient, accountClient, eventPublisher, validate, log)
	orderHandler := handlers.NewOrderHandler(orderService, log)

	// midlleware
	authMiddleware := customMiddleware.AuthMiddleware(authClientWrapper, cfg.Server.JWTSecret, cfg.Server.Audience, log)

	// Setup Server Web
	e := echo.New()
	e.Use(middleware.RequestID())
	e.Use(customMiddleware.LoggingMiddleware(log))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	routes.InitRoutes(e, orderHandler, authMiddleware)

	if err := e.Start(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func createGrpcConnection(url string, log *logrus.Logger) *grpc.ClientConn {
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC service at %s: %v", url, err)
	}

	return conn
}
