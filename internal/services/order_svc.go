package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/entities"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/helpers"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/messaging"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/db"
	apperrors "github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/errors"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/pkg/redis"
	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/repositories"

	accountpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/account"
	productpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/product"
)

type OrderService interface {
	// Core CRUD operations
	CreateOrder(ctx context.Context, userID uuid.UUID, req models.OrderDetailReq) (*entities.Order, error)
	GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]entities.OrderDetails, error)
	GetOrderItemsByOrderID(ctx context.Context, ID uuid.UUID) ([]entities.OrderItem, error)

	// Major lifecycle actions (Enterprise-grade)
	MarkOrderAsPaid(ctx context.Context, req models.MarkOrderAsPaidReq) (*entities.Order, error)
	MarkOrderAsShipped(ctx context.Context, req models.MarkOrderAsShippedReq) (*entities.Order, error)
	MarkOrderAsDelivered(ctx context.Context, req models.MarkOrderAsDeliveredReq) (*entities.Order, error)
	CancelOrder(ctx context.Context, orderID uuid.UUID, userID uuid.UUID) (*entities.Order, error)

	// Internal/Admin only
	UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, newStatus messaging.OrderStatus) (*entities.Order, error)
	ResetAllOrderCaches(ctx context.Context) error
}

type orderServiceImpl struct {
	orderRepo      repositories.OrderRepository
	redisClient    *redis.RedisClient
	productClient  productpb.ProductServiceClient
	accountClient  accountpb.AccountServiceClient
	eventPublisher *messaging.EventPublisherImpl
	validator      *validator.Validate
	log            *logrus.Logger
}

func NewOrderService(
	orderRepo repositories.OrderRepository,
	redisClient *redis.RedisClient,
	productClient productpb.ProductServiceClient,
	accountClient accountpb.AccountServiceClient,
	eventPublisher *messaging.EventPublisherImpl,
	validator *validator.Validate,
	log *logrus.Logger,
) OrderService {
	return &orderServiceImpl{
		orderRepo:      orderRepo,
		redisClient:    redisClient,
		productClient:  productClient,
		accountClient:  accountClient,
		eventPublisher: eventPublisher,
		validator:      validator,
		log:            log,
	}
}

type ItemWithProductAndSeller interface {
	db.GetOrderItemsByOrderIDRow | db.GetOrderItemsByOrderIDsRow
}

type OrderSource interface {
	db.GetOrdersByUserIDRow | db.Order | db.GetOrderByIDRow
}

func (s *orderServiceImpl) CreateOrder(ctx context.Context, userID uuid.UUID, req models.OrderDetailReq) (*entities.Order, error) {
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %s", apperrors.ErrInvalidRequestPayload, err.Error())
	}

	var productIDs []string
	for _, item := range req.Items {
		productIDs = append(productIDs, item.ID)
	}

	productDetailsMap, err := s.fetchProductDetails(ctx, productIDs)
	if err != nil {
		s.log.WithError(err).Error("Failed to fetch product details via gRPC")
		return nil, fmt.Errorf("failed to get product details: %w", err)
	}

	var totalPrice float64
	var itemsParams []db.CreateOrderItemParams
	pbStockItems := make([]*productpb.StockItem, 0, len(req.Items))

	for _, itemReq := range req.Items {
		productDetail, ok := productDetailsMap[itemReq.ID]
		if !ok {
			return nil, fmt.Errorf("product not found with ID: %s", itemReq.ID)
		}

		if itemReq.Quantity <= 0 {
			return nil, apperrors.ErrInvalidQuantity
		}

		/*
		* PENTING: TIDAK memeriksa stok di sini (if item.Quantity > productDetail.Stock).
		*Itu adalah "Check-Then-Act" (race condition). serahkan ke gRPC DecreaseStock.
		 */

		productID_uuid, _ := uuid.Parse(productDetail.Id)
		sellerID_uuid, _ := uuid.Parse(productDetail.SellerId)
		itemPrice := float64(productDetail.Price)
		totalPrice += itemPrice * float64(itemReq.Quantity)

		itemsParams = append(itemsParams, db.CreateOrderItemParams{
			ID:          uuid.New(),
			ProductID:   productID_uuid,
			SellerID:    sellerID_uuid,
			Quantity:    itemReq.Quantity,
			ProductName: productDetail.Name,
			Price:       fmt.Sprintf("%.2f", itemPrice),
			Description: helpers.StringToNullString(itemReq.Description),
		})

		pbStockItems = append(pbStockItems, &productpb.StockItem{
			ProductId:          itemReq.ID,
			QuantityToDecrease: itemReq.Quantity,
		})
	}

	orderParams := db.CreateOrderParams{
		ID:              uuid.New(),
		UserID:          userID,
		TotalPrice:      fmt.Sprintf("%.2f", totalPrice),
		ShippingAddress: req.Order.ShippingAddress,
		ShippingMethod:  req.Order.ShippingMethod,
		PaymentMethod:   req.Order.PaymentMethod,
		// ShippingTrackingCode: sql.NullString{String: req.Order.ShippingTrackingCode, Valid: req.Order.ShippingTrackingCode != ""},
		// PaymentGatewayID:     sql.NullString{String: req.Order.PaymentGatewayID, Valid: req.Order.PaymentGatewayID != ""},
	}

	tx, err := s.orderRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer tx.Rollback() // Rollback

	dbOrder, err := s.orderRepo.CreateOrder(ctx, tx, orderParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create order record: %w", err)
	}

	for _, itemParam := range itemsParams {
		itemParam.OrderID = dbOrder.ID
		if _, err := s.orderRepo.CreateOrderItem(ctx, tx, itemParam); err != nil {
			return nil, fmt.Errorf("failed to create order item record: %w", err)
		}
	}

	_, err = s.productClient.DecreaseStock(ctx, &productpb.DecreaseStockRequest{
		Items: pbStockItems,
	})

	if err != nil {
		st, _ := status.FromError(err)
		if st.Code() == codes.FailedPrecondition {
			return nil, apperrors.ErrProductOutOfStock
		}

		return nil, fmt.Errorf("gRPC call to catalog service failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit db transaction: %w", err)
	}

	// Publish order created event asynchronously
	go func() {
		event := messaging.OrderCreatedEvent{
			OrderID:       dbOrder.ID.String(),
			UserID:        userID.String(),
			TotalAmount:   totalPrice,
			ItemCount:     len(req.Items),
			PaymentMethod: req.Order.PaymentMethod,
			CreatedAt:     time.Now(),
		}

		if err := s.eventPublisher.PublishOrderCreated(context.Background(), event); err != nil {
			s.log.WithError(err).Warn("Failed to publish order created event")
		}
	}()

	s.InvalidateCachesForOrderChange(ctx, userID, dbOrder.ID)

	s.log.Infof("Order %s successfully created", dbOrder.ID)
	return toDomainOrder(dbOrder), nil
}

func (s *orderServiceImpl) GetOrdersByUserID(ctx context.Context, userID uuid.UUID) ([]entities.OrderDetails, error) {
	logger := s.log.WithField("user_id", userID)
	logger.Info("Retrieving order history for user")

	// Cache
	cacheKey := fmt.Sprintf("orders_user:%s", userID.String())
	val, err := s.redisClient.Client.Get(ctx, cacheKey).Result()
	if err == nil {
		var orderDetailsList []entities.OrderDetails
		if json.Unmarshal([]byte(val), &orderDetailsList) == nil {
			logger.Info("Cache HIT for user order history.")
			return orderDetailsList, nil
		}
	}
	logger.Info("Cache MISS for user order history.")

	// DB
	dbOrders, err := s.orderRepo.GetOrdersByUserID(ctx, userID)
	if err != nil {
		logger.WithError(err).Error("Failed to get orders from repository")
		return nil, fmt.Errorf("gagal mengambil data order: %w", err)
	}

	if len(dbOrders) == 0 {
		logger.Info("No orders found for this user.")
		return []entities.OrderDetails{}, nil
	}

	orderIDs := make([]uuid.UUID, len(dbOrders))
	for i, order := range dbOrders {
		orderIDs[i] = order.ID
	}

	dbOrderItems, err := s.orderRepo.GetOrderItemsByOrderIDs(ctx, orderIDs)
	if err != nil {
		logger.WithError(err).Error("Failed to get order items from repository")
		return nil, fmt.Errorf("gagal mengambil data item order: %w", err)
	}

	// ENRICHMENT
	productIDSet := make(map[string]bool)
	sellerIDSet := make(map[string]bool)
	for _, item := range dbOrderItems {
		productIDSet[item.ProductID.String()] = true
		sellerIDSet[item.SellerID.String()] = true
	}

	productIDs := make([]string, 0, len(productIDSet))
	for id := range productIDSet {
		productIDs = append(productIDs, id)
	}
	sellerIDs := make([]string, 0, len(sellerIDSet))
	for id := range sellerIDSet {
		sellerIDs = append(sellerIDs, id)
	}

	// gRPC
	productDetailsMap := make(map[string]*productpb.Product)
	accountDetailsMap := make(map[string]*accountpb.User)

	if len(productIDs) > 0 {
		productDetailsMap, err = s.fetchProductDetails(ctx, productIDs)
		if err != nil {
			return nil, err
		}

		accountDetailsMap, err = s.fetchAccountDetails(ctx, sellerIDs)
		if err != nil {
			return nil, err
		}
	}

	itemsByOrderID := make(map[uuid.UUID][]db.GetOrderItemsByOrderIDsRow)
	for _, item := range dbOrderItems {
		itemsByOrderID[item.OrderID] = append(itemsByOrderID[item.OrderID], item)
	}

	finalOrderDetailsList := make([]entities.OrderDetails, 0, len(dbOrders))

	for _, dbOrder := range dbOrders {

		assembledItems := make([]entities.OrderItem, 0)

		if dbItemsInThisOrder, ok := itemsByOrderID[dbOrder.ID]; ok {

			for _, dbItem := range dbItemsInThisOrder {
				productDetail := productDetailsMap[dbItem.ProductID.String()]
				accountDetail := accountDetailsMap[dbItem.SellerID.String()]

				domainItem := toDomainOrderItem(dbItem, productDetail, accountDetail)
				assembledItems = append(assembledItems, domainItem)
			}
		}

		finalOrderDetailsList = append(finalOrderDetailsList, entities.OrderDetails{
			Order: *toDomainOrder(dbOrder),
			Items: assembledItems,
		})
	}

	jsonBytes, err := json.Marshal(finalOrderDetailsList)
	if err == nil {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.redisClient.Client.Set(cacheCtx, cacheKey, jsonBytes, 5*time.Minute).Err(); err != nil {
				s.log.WithField("key", cacheKey).Warn("Gagal menyimpan hasil ke cache")
			}
		}()
	}

	return finalOrderDetailsList, nil
}

func (s *orderServiceImpl) GetOrderItemsByOrderID(ctx context.Context, ID uuid.UUID) ([]entities.OrderItem, error) {
	if ID == uuid.Nil {
		return []entities.OrderItem{}, nil
	}

	// Redis Cache
	cacheKey := fmt.Sprintf("order_items:%s", ID.String())
	val, err := s.redisClient.Client.Get(ctx, cacheKey).Result()
	if err == nil {
		var cachedItems []entities.OrderItem
		if json.Unmarshal([]byte(val), &cachedItems) == nil {
			return cachedItems, nil
		}
	}

	dbOrderItems, err := s.orderRepo.GetOrderItemsByOrderID(ctx, ID)
	if err != nil {
		return nil, err
	}

	productIDs, sellerIDs := collectIDsForEnrichment(dbOrderItems)

	// gRPC Product
	productDetailsMap, err := s.fetchProductDetails(ctx, productIDs)
	if err != nil {
		return nil, err
	}

	//gRPC Account
	accountDetailsMap, err := s.fetchAccountDetails(ctx, sellerIDs)
	if err != nil {
		return nil, err
	}

	finalItems := s.assembleOrderItems(dbOrderItems, productDetailsMap, accountDetailsMap)

	// Save to Redis
	jsonBytes, err := json.Marshal(finalItems)
	if err == nil {
		if err := s.redisClient.Client.Set(ctx, cacheKey, jsonBytes, 10*time.Minute).Err(); err != nil {
			s.log.Printf("Gagal menyimpan kembali ke cache: %v", err)
		}
	}

	return finalItems, nil

}

func (s *orderServiceImpl) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, newStatus messaging.OrderStatus) (*entities.Order, error) {
	order, err := s.orderRepo.UpdateOrderStatus(ctx, orderID, string(newStatus))
	if err != nil {
		return nil, err
	}

	// Publish status changed event asynchronously
	go func() {
		totalPrice, err := helpers.StringToFloat64(order.TotalPrice)
		if err != nil {
			s.log.WithError(err).Warn("Failed to convert total price to float64")
			totalPrice = 0
		}

		event := messaging.OrderStatusChangedEvent{
			OrderID:     orderID.String(),
			UserID:      order.UserID.String(),
			Email:       "", // TODO: Get from user service if needed
			OldStatus:   "", // TODO: Track old status if needed
			NewStatus:   newStatus,
			TotalAmount: totalPrice,
			ChangedAt:   time.Now(),
		}

		if err := s.eventPublisher.PublishOrderStatusChanged(context.Background(), event); err != nil {
			s.log.WithError(err).Warn("Failed to publish order status event")
		}
	}()

	return toDomainOrder(order), nil
}

func (s *orderServiceImpl) CancelOrder(ctx context.Context, orderID uuid.UUID, userID uuid.UUID) (*entities.Order, error) {

	tx, err := s.orderRepo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer tx.Rollback() // Rollback

	itemsToRestock, err := s.orderRepo.GetItemsForRestock(ctx, tx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get items for restock: %w", err)
	}

	dbOrder, err := s.orderRepo.CancelOrder(ctx, tx, orderID, userID)
	if err != nil {
		return nil, err
	}

	pbStockItems := make([]*productpb.StockItem, len(itemsToRestock))
	for i, item := range itemsToRestock {
		pbStockItems[i] = &productpb.StockItem{
			ProductId:          item.ProductID.String(),
			QuantityToDecrease: item.Quantity,
		}
	}

	grpcCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := s.productClient.IncreaseStock(grpcCtx, &productpb.IncreaseStockRequest{
		Items: pbStockItems,
	}); err != nil {
		return nil, fmt.Errorf("gRPC call to catalog service failed, order cancellation rolled back: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit db transaction: %w", err)
	}

	s.InvalidateCachesForOrderChange(ctx, userID, dbOrder.ID)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		totalPrice, err := helpers.StringToFloat64(dbOrder.TotalPrice)
		if err != nil {
			s.log.WithError(err).Warn("Failed to convert total price to float64")
			totalPrice = 0
		}

		// Use specific OrderCancelledEvent instead of generic OrderStatusChangedEvent
		event := messaging.OrderCancelledEvent{
			OrderID:         dbOrder.ID.String(),
			UserID:          dbOrder.UserID.String(),
			CancelledBy:     "user",                        // User-initiated cancellation
			CancelReason:    "User requested cancellation", // TODO: Get from request if available
			CancelCategory:  "user_request",
			RefundAmount:    totalPrice,
			RefundMethod:    "original_payment",          // TODO: Determine refund method
			CancellationFee: 0,                           // No penalty for now
			OriginalStatus:  string(dbOrder.OrderStatus), // Status before cancellation
			CancelledAt:     time.Now(),
		}

		if err := s.eventPublisher.PublishOrderCancelled(ctx, event); err != nil {
			s.log.Errorf("Failed to publish order cancelled event: %v", err)
		}
	}()

	return toDomainOrder(dbOrder), nil
}

// MarkOrderAsPaid handles payment confirmation with comprehensive validation
func (s *orderServiceImpl) MarkOrderAsPaid(ctx context.Context, req models.MarkOrderAsPaidReq) (*entities.Order, error) {
	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %s", apperrors.ErrInvalidRequestPayload, err.Error())
	}

	// Verify order exists and is in correct status
	dbOrder, err := s.orderRepo.GetOrderByID(ctx, req.OrderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// Business rule: Order must be pending to be marked as paid
	if dbOrder.OrderStatus != "pending" {
		return nil, fmt.Errorf("order must be in 'pending' status to mark as paid, current status: %s", dbOrder.OrderStatus)
	}

	// Update order status to paid
	updatedOrder, err := s.orderRepo.UpdateOrderStatus(ctx, req.OrderID, "paid")
	if err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	// Invalidate caches
	userID, _ := uuid.Parse(updatedOrder.UserID.String())
	s.InvalidateCachesForOrderChange(ctx, userID, updatedOrder.ID)

	// Publish OrderPaidEvent asynchronously
	go func() {
		ctxPublish, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		event := messaging.OrderPaidEvent{
			OrderID:        updatedOrder.ID.String(),
			UserID:         updatedOrder.UserID.String(),
			PaidAmount:     req.PaidAmount,
			PaymentMethod:  req.PaymentMethod,
			PaymentGateway: req.PaymentGateway,
			TransactionID:  req.TransactionID,
			PaymentProof:   req.PaymentProof,
			OriginalAmount: req.OriginalAmount,
			DiscountAmount: req.DiscountAmount,
			PaidAt:         time.Now(),
		}

		if err := s.eventPublisher.PublishOrderPaid(ctxPublish, event); err != nil {
			s.log.Errorf("Failed to publish order paid event: %v", err)
		}
	}()

	s.log.Infof("Order %s marked as paid (Payment Gateway: %s, TX: %s)",
		updatedOrder.ID, req.PaymentGateway, req.TransactionID)

	return toDomainOrder(updatedOrder), nil
}

// MarkOrderAsShipped handles shipment initiation with tracking information
func (s *orderServiceImpl) MarkOrderAsShipped(ctx context.Context, req models.MarkOrderAsShippedReq) (*entities.Order, error) {
	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %s", apperrors.ErrInvalidRequestPayload, err.Error())
	}

	// Verify order exists
	dbOrder, err := s.orderRepo.GetOrderByID(ctx, req.OrderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// Business rule: Order must be paid before shipping
	if dbOrder.OrderStatus != "paid" {
		return nil, fmt.Errorf("order must be 'paid' before shipping, current status: %s", dbOrder.OrderStatus)
	}

	// Update order status to shipped
	updatedOrder, err := s.orderRepo.UpdateOrderStatus(ctx, req.OrderID, "shipped")
	if err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	// Invalidate caches
	userID, _ := uuid.Parse(updatedOrder.UserID.String())
	s.InvalidateCachesForOrderChange(ctx, userID, updatedOrder.ID)

	// Parse estimated arrival
	estimatedArrival, err := time.Parse(time.RFC3339, req.EstimatedArrival)
	if err != nil {
		// If parsing fails, default to 3 days from now
		estimatedArrival = time.Now().Add(3 * 24 * time.Hour)
		s.log.Warnf("Failed to parse estimated arrival, using default (3 days): %v", err)
	}

	// Publish OrderShippedEvent asynchronously
	go func() {
		ctxPublish, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		event := messaging.OrderShippedEvent{
			OrderID:          updatedOrder.ID.String(),
			UserID:           updatedOrder.UserID.String(),
			TrackingNumber:   req.TrackingNumber,
			Courier:          req.Courier,
			WarehouseID:      req.WarehouseID,
			PackageWeight:    req.PackageWeight,
			ShippingCost:     req.ShippingCost,
			EstimatedArrival: estimatedArrival,
			ShippedAt:        time.Now(),
		}

		if err := s.eventPublisher.PublishOrderShipped(ctxPublish, event); err != nil {
			s.log.Errorf("Failed to publish order shipped event: %v", err)
		}
	}()

	s.log.Infof("Order %s marked as shipped (Courier: %s, Tracking: %s)",
		updatedOrder.ID, req.Courier, req.TrackingNumber)

	return toDomainOrder(updatedOrder), nil
}

// MarkOrderAsDelivered handles successful delivery confirmation
func (s *orderServiceImpl) MarkOrderAsDelivered(ctx context.Context, req models.MarkOrderAsDeliveredReq) (*entities.Order, error) {
	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("%w: %s", apperrors.ErrInvalidRequestPayload, err.Error())
	}

	// Verify order exists
	dbOrder, err := s.orderRepo.GetOrderByID(ctx, req.OrderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// Business rule: Order must be shipped before delivery
	if dbOrder.OrderStatus != "shipped" {
		return nil, fmt.Errorf("order must be 'shipped' before delivery, current status: %s", dbOrder.OrderStatus)
	}

	// Update order status to delivered
	updatedOrder, err := s.orderRepo.UpdateOrderStatus(ctx, req.OrderID, "delivered")
	if err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	// Invalidate caches
	userID, _ := uuid.Parse(updatedOrder.UserID.String())
	s.InvalidateCachesForOrderChange(ctx, userID, updatedOrder.ID)

	// Publish OrderDeliveredEvent asynchronously
	go func() {
		ctxPublish, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		event := messaging.OrderDeliveredEvent{
			OrderID:       updatedOrder.ID.String(),
			UserID:        updatedOrder.UserID.String(),
			ReceiverName:  req.ReceiverName,
			DeliveryProof: req.DeliveryProof,
			DeliveryNotes: req.DeliveryNotes,
			DeliveredAt:   time.Now(),
			ShippedAt:     updatedOrder.UpdatedAt, // Approximation
		}

		if err := s.eventPublisher.PublishOrderDelivered(ctxPublish, event); err != nil {
			s.log.Errorf("Failed to publish order delivered event: %v", err)
		}
	}()

	s.log.Infof("Order %s marked as delivered (Receiver: %s)",
		updatedOrder.ID, req.ReceiverName)

	return toDomainOrder(updatedOrder), nil
}

// ------- HELPERS -------

func collectIDsForEnrichment[T ItemWithProductAndSeller](items []T) (productIDs []string, sellerIDs []string) {
	productIDMap := make(map[string]bool)
	sellerIDMap := make(map[string]bool)

	for _, item := range items {
		v := reflect.ValueOf(item)

		productID := v.FieldByName("ProductID").Interface().(uuid.UUID)
		sellerID := v.FieldByName("SellerID").Interface().(uuid.UUID)

		productIDMap[productID.String()] = true
		sellerIDMap[sellerID.String()] = true
	}

	for id := range productIDMap {
		productIDs = append(productIDs, id)
	}
	for id := range sellerIDMap {
		sellerIDs = append(sellerIDs, id)
	}
	return
}

func (s *orderServiceImpl) fetchProductDetails(ctx context.Context, productIDs []string) (map[string]*productpb.Product, error) {
	productsResponse, err := s.productClient.GetProducts(ctx, &productpb.GetProductsRequest{Ids: productIDs})
	if err != nil {
		return nil, fmt.Errorf("failed to get product details via gRPC: %w", err)
	}

	productDetailsMap := make(map[string]*productpb.Product)
	for _, p := range productsResponse.Products {
		productDetailsMap[p.Id] = p
	}

	return productDetailsMap, nil
}

func (s *orderServiceImpl) fetchAccountDetails(ctx context.Context, sellerIDs []string) (map[string]*accountpb.User, error) {
	accountResponse, err := s.accountClient.GetUsers(ctx, &accountpb.GetUsersRequest{Ids: sellerIDs})
	if err != nil {
		return nil, fmt.Errorf("failed to get account details via gRPC: %w", err)
	}

	accountDetailsMap := make(map[string]*accountpb.User)
	for _, a := range accountResponse.Users {
		accountDetailsMap[a.Id] = a
	}

	return accountDetailsMap, nil
}

func (s *orderServiceImpl) assembleOrderItems(
	dbOrderItems []db.GetOrderItemsByOrderIDRow,
	productDetailsMap map[string]*productpb.Product,
	accountDetailsMap map[string]*accountpb.User,
) []entities.OrderItem {

	enrichedItems := make([]entities.OrderItem, 0, len(dbOrderItems))

	for _, dbItem := range dbOrderItems {
		productDetail, okP := productDetailsMap[dbItem.ProductID.String()]
		if !okP {
			s.log.WithField("product_id", dbItem.ProductID).Warn("Product not found during assembly, item skipped.")
			continue
		}

		accountDetail, okA := accountDetailsMap[productDetail.SellerId]
		if !okA {
			s.log.WithField("seller_id", productDetail.SellerId).Warn("Account not found during assembly, item skipped.")
			continue
		}

		priceAtPurchase, _ := strconv.ParseFloat(dbItem.Price, 64)
		var description string
		if dbItem.Description.Valid {
			description = dbItem.Description.String
		}

		enrichedItem := entities.OrderItem{
			ID:              dbItem.ID.String(),
			OrderID:         dbItem.OrderID.String(),
			SellerID:        productDetail.SellerId,
			SellerName:      accountDetail.Name,
			ProductID:       dbItem.ProductID.String(),
			ProductName:     productDetail.Name,
			Quantity:        int(dbItem.Quantity),
			ProductPrice:    priceAtPurchase,
			CartDescription: description,
		}
		enrichedItems = append(enrichedItems, enrichedItem)
	}

	return enrichedItems
}

func toDomainOrder[T OrderSource](dbOrder T) *entities.Order {
	v := reflect.ValueOf(dbOrder)

	id := v.FieldByName("ID").Interface().(uuid.UUID)
	userID := v.FieldByName("UserID").Interface().(uuid.UUID)
	orderStatus := v.FieldByName("OrderStatus").Interface().(db.OrderStatus)
	totalPriceStr := v.FieldByName("TotalPrice").Interface().(string)
	shippingAddress := v.FieldByName("ShippingAddress").Interface().(string)
	shippingMethod := v.FieldByName("ShippingMethod").Interface().(string)
	shippingTrackingCode := v.FieldByName("ShippingTrackingCode").Interface().(sql.NullString)
	paymentMethod := v.FieldByName("PaymentMethod").Interface().(string)
	paymentGatewayID := v.FieldByName("PaymentGatewayID").Interface().(sql.NullString)
	createdAt := v.FieldByName("CreatedAt").Interface().(time.Time)
	updatedAt := v.FieldByName("UpdatedAt").Interface().(time.Time)

	totalPrice, _ := strconv.ParseFloat(strings.TrimSpace(totalPriceStr), 64)
	var trackingCode string
	if shippingTrackingCode.Valid {
		trackingCode = shippingTrackingCode.String
	}
	var gatewayID string
	if paymentGatewayID.Valid {
		gatewayID = paymentGatewayID.String
	}

	return &entities.Order{
		ID:                   id,
		UserID:               userID,
		OrderStatus:          string(orderStatus),
		TotalPrice:           totalPrice,
		ShippingAddress:      shippingAddress,
		ShippingMethod:       shippingMethod,
		ShippingTrackingCode: trackingCode,
		PaymentMethod:        paymentMethod,
		PaymentGatewayID:     gatewayID,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
	}
}

func toDomainOrderItem(dbItem db.GetOrderItemsByOrderIDsRow, product *productpb.Product, seller *accountpb.User) entities.OrderItem {
	price, _ := strconv.ParseFloat(dbItem.Price, 64)

	var productName, sellerName string

	if product != nil {
		productName = product.Name
	}
	if seller != nil {
		sellerName = seller.Name
	}

	return entities.OrderItem{
		ID:              dbItem.ID.String(),
		OrderID:         dbItem.OrderID.String(),
		SellerID:        dbItem.SellerID.String(),
		SellerName:      sellerName,
		ProductID:       dbItem.ProductID.String(),
		ProductName:     productName,
		Quantity:        int(dbItem.Quantity),
		ProductPrice:    price,
		CartDescription: dbItem.Description.String,
	}
}

func (s *orderServiceImpl) InvalidateCachesForOrderChange(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) {
	userCacheKey := fmt.Sprintf("orders_user:%s", userID.String())
	detailCacheKey := fmt.Sprintf("order_items:%s", orderID.String())

	keysToDel := []string{userCacheKey, detailCacheKey}

	s.log.Infof("Invalidating targeted order caches: %v", keysToDel)

	// agar tidak memblokir
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.redisClient.Client.Del(cacheCtx, keysToDel...).Err(); err != nil {
			s.log.Warnf("Failed to invalidate order caches for user %s: %v", userID, err)
		}
	}()
}

func (s *orderServiceImpl) ResetAllOrderCaches(ctx context.Context) error {
	s.log.Info("Starting to reset ALL order caches...")

	resetCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	patterns := []string{
		"orders_user:*",
		"order_items:*",
	}

	var totalKeysDeleted int64 = 0

	for _, pattern := range patterns {
		s.log.Infof("Scanning and deleting keys matching pattern: %s", pattern)

		var cursor uint64
		var keysFoundInPattern int64 = 0

		for {
			keys, nextCursor, err := s.redisClient.Client.Scan(resetCtx, cursor, pattern, 100).Result()
			if err != nil {
				s.log.Errorf("Error during Redis SCAN with pattern '%s': %v", pattern, err)
				return err
			}

			if len(keys) > 0 {
				if err := s.redisClient.Client.Del(resetCtx, keys...).Err(); err != nil {
					s.log.Warnf("Failed to delete batch of %d keys: %v", len(keys), err)
				}
				keysFoundInPattern += int64(len(keys))
			}

			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}
		s.log.Infof("Successfully deleted %d keys matching pattern '%s'", keysFoundInPattern, pattern)
		totalKeysDeleted += keysFoundInPattern
	}

	s.log.Infof("Successfully reset a total of %d order cache keys.", totalKeysDeleted)
	return nil
}
