package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/mafi020/ecom-golang-micro/internal/apperrors"
	"github.com/mafi020/ecom-golang-micro/internal/entity"
	"github.com/mafi020/ecom-golang-micro/internal/utils"
	"github.com/mafi020/ecom-golang-micro/pkg/events"
	orderpb "github.com/mafi020/ecom-golang-micro/proto/order"
)

type paymentRepo interface {
	Create(ctx context.Context, payment *entity.Payment) (*entity.Payment, error)
	CreateOnlineTransaction(ctx context.Context, tx *entity.OnlineTransaction) (*entity.OnlineTransaction, error)
	CreateCODDetail(ctx context.Context, detail *entity.CODDetail) (*entity.CODDetail, error)
	GetPaymentByOrderID(ctx context.Context, orderID int64) (*entity.Payment, error)
	UpdateStatus(ctx context.Context, id int64, status entity.PaymentStatus) error
	UpdateCODDetail(ctx context.Context, paymentID int64, detail *entity.CODDetail) (*entity.CODDetail, error)
}

type tm interface {
	WithinTransaction(context.Context, func(context.Context) error) error
}

type PaymentUseCase struct {
	paymentRepo paymentRepo
	orderClient orderpb.OrderServiceClient
	tm          tm
	broker      *utils.EventBroker
}

func NewPaymentUseCase(paymentRepo paymentRepo, orderClient orderpb.OrderServiceClient, tm tm, broker *utils.EventBroker) *PaymentUseCase {
	return &PaymentUseCase{
		paymentRepo: paymentRepo,
		orderClient: orderClient,
		tm:          tm,
		broker:      broker,
	}
}

// ── Online Payment ────────────────────────────────────────────────────────────

func (uc *PaymentUseCase) PayOnline(
	ctx context.Context,
	userID int64,
	orderID int64,
	provider entity.PaymentProvider,
	gatewayRef string,
	gatewayStatus string,
	rawResponse []byte,
) (*entity.Payment, error) {
	// 1. Fetch Order over network via gRPC instead of local DB query
	orderResp, err := uc.orderClient.GetOrder(ctx, &orderpb.GetOrderRequest{
		Id:     orderID,
		UserId: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order via grpc: %w", err)
	}

	orderProto := orderResp.GetOrder()
	if orderProto.GetStatus().String() != string(entity.OrderStatusPending) {
		return nil, &apperrors.ValidationError{Errors: map[string]string{
			"status": fmt.Sprintf("order must be pending for online payment, current: %s", orderProto.GetStatus()),
		}}
	}

	var payment *entity.Payment

	// 2. Perform local payment operations inside a local atomic transaction
	err = uc.tm.WithinTransaction(ctx, func(txCtx context.Context) error {
		txUUID := uuid.New().String()
		payment, err = uc.paymentRepo.Create(txCtx, &entity.Payment{
			OrderID:       orderProto.GetId(),
			TransactionID: txUUID,
			Method:        entity.PaymentMethodOnline,
			Status:        entity.PaymentStatusCompleted,
			AmountCents:   orderProto.GetTotalPrice(), // Uses gRPC response price field
		})
		if err != nil {
			return err
		}

		onlineTx, err := uc.paymentRepo.CreateOnlineTransaction(txCtx, &entity.OnlineTransaction{
			PaymentID:     payment.ID,
			Provider:      provider,
			GatewayRef:    gatewayRef,
			GatewayStatus: gatewayStatus,
			RawResponse:   rawResponse,
		})
		if err != nil {
			return err
		}
		payment.OnlineTransaction = onlineTx
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 3. emit "paymet.completed" event
	paymentEvent := events.PaymentCompletedEvent{
		OrderID:       payment.OrderID,
		UserID:        userID,
		TransactionID: payment.TransactionID,
		AmountCents:   payment.AmountCents,
	}

	// Broadcast token safely onto the network queue channel
	if err := uc.broker.Publish(ctx, "payment.completed", paymentEvent); err != nil {
		// Log the warning, but return success to client because their ledger is already secure in DB
		slog.Error("Reconciliation database record secured, but event broadcast missed", slog.Any("error", err))
	}

	return payment, nil
}

// ── COD Payment ───────────────────────────────────────────────────────────────

func (uc *PaymentUseCase) PayCOD(ctx context.Context, userID, orderID int64) (*entity.Payment, error) {
	// Fetch Order details over network via gRPC
	orderResp, err := uc.orderClient.GetOrder(ctx, &orderpb.GetOrderRequest{
		Id:     orderID,
		UserId: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order via grpc: %w", err)
	}

	orderProto := orderResp.GetOrder()
	if orderProto.GetStatus().String() != string(entity.OrderStatusPending) {
		return nil, &apperrors.ValidationError{Errors: map[string]string{
			"status": fmt.Sprintf("order must be pending for COD, current: %s", orderProto.GetStatus()),
		}}
	}

	var payment *entity.Payment

	err = uc.tm.WithinTransaction(ctx, func(txCtx context.Context) error {
		txUUID := uuid.New().String()
		payment, err = uc.paymentRepo.Create(txCtx, &entity.Payment{
			OrderID:       orderProto.GetId(),
			TransactionID: txUUID,
			Method:        entity.PaymentMethodCOD,
			Status:        entity.PaymentStatusPending,
			AmountCents:   orderProto.GetTotalPrice(),
		})
		if err != nil {
			return err
		}

		codDetail, err := uc.paymentRepo.CreateCODDetail(txCtx, &entity.CODDetail{
			PaymentID: payment.ID,
		})
		if err != nil {
			return err
		}
		payment.CODDetail = codDetail
		return nil
	})

	return payment, err
}

// ── COD Collection (admin) ────────────────────────────────────────────────────

func (uc *PaymentUseCase) CollectCOD(ctx context.Context, orderID int64) (*entity.Payment, error) {
	payment, err := uc.paymentRepo.GetPaymentByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if payment.Method != entity.PaymentMethodCOD {
		return nil, &apperrors.ValidationError{Errors: map[string]string{
			"method": "payment is not COD",
		}}
	}

	if payment.Status == entity.PaymentStatusCompleted {
		return nil, &apperrors.ValidationError{Errors: map[string]string{
			"status": "payment already completed",
		}}
	}

	now := time.Now()

	err = uc.tm.WithinTransaction(ctx, func(txCtx context.Context) error {
		codDetail, err := uc.paymentRepo.UpdateCODDetail(txCtx, payment.ID, &entity.CODDetail{
			CollectedAt: &now,
		})
		if err != nil {
			return err
		}
		payment.CODDetail = codDetail

		if err := uc.paymentRepo.UpdateStatus(txCtx, payment.ID, entity.PaymentStatusCompleted); err != nil {
			return err
		}
		payment.Status = entity.PaymentStatusCompleted
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Emit "payment.completed" event
	// Progress order state machine remotely (shipped -> delivered)
	paymentEvent := events.PaymentCompletedEvent{
		OrderID:       payment.OrderID,
		TransactionID: payment.TransactionID,
		AmountCents:   payment.AmountCents,
	}

	// paymentEvent := events.PaymentCompletedEvent{
	// 	OrderID:       payment.OrderID,
	// 	UserID:        userID,
	// 	TransactionID: payment.TransactionID,
	// 	AmountCents:   payment.AmountCents,
	// }

	// Broadcast transaction reconciliation event so order transitions to PAID/DELIVERED asynchronously
	if err := uc.broker.Publish(ctx, "payment.completed", paymentEvent); err != nil {
		slog.Error("COD collected locally, but async event distribution missed", slog.Any("error", err))
	}

	return payment, nil
}

func (uc *PaymentUseCase) GetPaymentByOrderID(ctx context.Context, orderID, userID int64) (*entity.Payment, error) {
	// Authenticate order validation check via gRPC network layer boundary
	_, err := uc.orderClient.GetOrder(ctx, &orderpb.GetOrderRequest{
		Id:     orderID,
		UserId: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("authorization check failed: order service unresolvable: %w", err)
	}

	return uc.paymentRepo.GetPaymentByOrderID(ctx, orderID)
}
