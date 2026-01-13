package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/db"
)

type OrderRepository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CreateOrder(ctx context.Context, tx *sql.Tx, params db.CreateOrderParams) (db.Order, error)
	CreateOrderItem(ctx context.Context, tx *sql.Tx, params db.CreateOrderItemParams) (db.OrderItem, error)
	GetOrderByID(ctx context.Context, orderID uuid.UUID) (db.GetOrderByIDRow, error)
	GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]db.GetOrdersByUserIDRow, error)
	GetOrderItemsByOrderID(ctx context.Context, orderIDs uuid.UUID) ([]db.GetOrderItemsByOrderIDRow, error)
	GetOrderItemsByOrderIDs(ctx context.Context, orderIDs []uuid.UUID) ([]db.GetOrderItemsByOrderIDsRow, error)
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string) (db.Order, error)
	CancelOrder(ctx context.Context, tx *sql.Tx, orderID uuid.UUID, userID uuid.UUID) (db.Order, error)
	GetItemsForRestock(ctx context.Context, tx *sql.Tx, orderID uuid.UUID) ([]db.GetItemsForRestockRow, error)
}

type orderRepository struct {
	db    *sql.DB
	q     *db.Queries
	store *db.Store
	log   *logrus.Logger
}

func NewOrderRepository(
	db *sql.DB,
	q *db.Queries,
	store *db.Store,
	log *logrus.Logger,

) OrderRepository {
	return &orderRepository{
		db:    db,
		q:     q,
		store: store,
		log:   log,
	}
}

func (r *orderRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *orderRepository) CreateOrder(ctx context.Context, tx *sql.Tx, params db.CreateOrderParams) (db.Order, error) {
	qtx := r.q.WithTx(tx)

	createdOrder, err := qtx.CreateOrder(ctx, params)
	if err != nil {
		r.log.WithField("error", err).Error("Failed to create a record order in the transaction")
		return db.Order{}, fmt.Errorf("failed to create a record order: %w", err)
	}

	return createdOrder, nil
}

func (r *orderRepository) CreateOrderItem(ctx context.Context, tx *sql.Tx, params db.CreateOrderItemParams) (db.OrderItem, error) {
	qtx := r.q.WithTx(tx)

	createdItem, err := qtx.CreateOrderItem(ctx, params)
	if err != nil {
		r.log.WithFields(logrus.Fields{
			"product_id": params.ProductID,
			"order_id":   params.OrderID,
			"error":      err,
		}).Error("Failed to create a record order item in the transaction")
		return db.OrderItem{}, fmt.Errorf("failed to create a record order item: %w", err)
	}

	return createdItem, nil
}

func (r *orderRepository) GetOrderByID(ctx context.Context, orderID uuid.UUID) (db.GetOrderByIDRow, error) {
	order, err := r.store.GetOrderByID(ctx, orderID)
	if err != nil {
		r.log.WithFields(logrus.Fields{"order_id": orderID, "error": err}).Error("Failed to receive order from DB")
		return db.GetOrderByIDRow{}, fmt.Errorf("failed to receive order from DB: %w", err)
	}
	return order, nil
}

func (r *orderRepository) GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]db.GetOrdersByUserIDRow, error) {
	orders, err := r.store.GetOrdersByUserID(ctx, userID)
	if err != nil {
		r.log.WithFields(logrus.Fields{"user_id": userID, "error": err}).Error("Failed to receive orders from DB")
		return nil, fmt.Errorf("failed to receive orders from DB: %w", err)
	}
	return orders, nil
}

func (r *orderRepository) GetOrderItemsByOrderIDs(ctx context.Context, orderIDs []uuid.UUID) ([]db.GetOrderItemsByOrderIDsRow, error) {
	items, err := r.store.GetOrderItemsByOrderIDs(ctx, orderIDs)
	if err != nil {
		r.log.WithFields(logrus.Fields{"order_ids": orderIDs, "error": err}).Error("Failed to receive order items from DB")
		return nil, fmt.Errorf("failed to receive order items from DB: %w", err)
	}
	return items, nil
}

func (r *orderRepository) GetOrderItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]db.GetOrderItemsByOrderIDRow, error) {
	items, err := r.store.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		r.log.WithFields(logrus.Fields{"order_ids": orderID, "error": err}).Error("Failed to retrieve order items from DB")
		return nil, fmt.Errorf("failed to retrieve order items from DB: %w", err)
	}
	return items, nil
}

func (r *orderRepository) GetItemsForRestock(ctx context.Context, tx *sql.Tx, orderID uuid.UUID) ([]db.GetItemsForRestockRow, error) {
	qtx := r.q.WithTx(tx)
	return qtx.GetItemsForRestock(ctx, orderID)
}

func (r *orderRepository) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status string) (db.Order, error) {
	order, err := r.q.UpdateOrderStatus(ctx, db.UpdateOrderStatusParams{
		ID:          orderID,
		OrderStatus: db.OrderStatus(status),
	})

	if err == sql.ErrNoRows {
		return db.Order{}, fmt.Errorf("order not found")
	}

	if err != nil {
		return db.Order{}, fmt.Errorf("failed to update order status: %w", err)
	}

	return order, nil
}

func (r *orderRepository) CancelOrder(ctx context.Context, tx *sql.Tx, orderID uuid.UUID, userID uuid.UUID) (db.Order, error) {
	qtx := r.q.WithTx(tx)
	dbOrder, err := qtx.CancelOrder(ctx, db.CancelOrderParams{
		ID:     orderID,
		UserID: userID,
	})
	if err == sql.ErrNoRows {
		return db.Order{}, fmt.Errorf("order not found")
	}

	if err != nil {
		return db.Order{}, fmt.Errorf("failed to cancel order: %w", err)
	}
	return dbOrder, nil
}
