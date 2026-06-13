package handler

import (
	"context"

	"github.com/mafi020/ecom-golang-micro/internal/delivery/gRPC/utils"
	"github.com/mafi020/ecom-golang-micro/internal/entity"
	paymentpb "github.com/mafi020/ecom-golang-micro/proto/payment"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type paymentUseCase interface {
	PayOnline(ctx context.Context, userID, orderID int64, provider entity.PaymentProvider, gatewayRef, gatewayStatus string, rawResponse []byte) (*entity.Payment, error)
	PayCOD(ctx context.Context, userID, orderID int64) (*entity.Payment, error)
	CollectCOD(ctx context.Context, orderID int64) (*entity.Payment, error)
	GetPaymentByOrderID(ctx context.Context, orderID, userID int64) (*entity.Payment, error)
}

type PaymentGRPCHandler struct {
	paymentpb.UnimplementedPaymentServiceServer
	uc paymentUseCase
}

func NewPaymentGRPCHandler(uc paymentUseCase) *PaymentGRPCHandler {
	return &PaymentGRPCHandler{uc: uc}
}

func (h *PaymentGRPCHandler) CreatePayment(ctx context.Context, req *paymentpb.CreatePaymentRequest) (*paymentpb.CreatePaymentResponse, error) {
	var payment *entity.Payment
	var err error

	switch req.GetMethod() {
	case paymentpb.PaymentMethod_PAYMENT_METHOD_ONLINE:
		var provider entity.PaymentProvider
		switch req.GetProvider() {
		case paymentpb.PaymentProvider_PAYMENT_PROVIDER_STRIPE:
			provider = entity.PaymentProviderStripe
		case paymentpb.PaymentProvider_PAYMENT_PROVIDER_SSLCOMMERZ:
			provider = entity.PaymentProviderSSLCOMMERZ
		case paymentpb.PaymentProvider_PAYMENT_PROVIDER_PAYPAL:
			provider = entity.PaymentProviderPaypal
		}

		payment, err = h.uc.PayOnline(
			ctx,
			req.GetUserId(),
			req.GetOrderId(),
			provider,
			req.GetGatewayRef(),
			req.GetGatewayStatus(),
			[]byte(req.GetRawResponse()),
		)

	case paymentpb.PaymentMethod_PAYMENT_METHOD_COD:
		payment, err = h.uc.PayCOD(ctx, req.GetUserId(), req.GetOrderId())

	default:
		payment, err = nil, utils.HandleGRPCError(context.Canceled)
	}

	if err != nil {
		return nil, utils.HandleGRPCError(err)
	}

	return &paymentpb.CreatePaymentResponse{
		Payment: h.toProtoPayment(payment),
	}, nil
}

func (h *PaymentGRPCHandler) GetPaymentByOrderId(ctx context.Context, req *paymentpb.GetPaymentRequest) (*paymentpb.Payment, error) {
	// 🚀 FIXED: Context fields passed seamlessly down into usecase authorization logic
	payment, err := h.uc.GetPaymentByOrderID(ctx, req.GetOrderId(), req.GetUserId())
	if err != nil {
		return nil, utils.HandleGRPCError(err)
	}
	return h.toProtoPayment(payment), nil
}

// 🚀 FIXED: Added administrative CollectCOD handling implementation block
func (h *PaymentGRPCHandler) CollectCOD(ctx context.Context, req *paymentpb.CollectCODRequest) (*paymentpb.CollectCODResponse, error) {
	payment, err := h.uc.CollectCOD(ctx, req.GetOrderId())
	if err != nil {
		return nil, utils.HandleGRPCError(err)
	}
	return &paymentpb.CollectCODResponse{
		Payment: h.toProtoPayment(payment),
	}, nil
}

// ── INTERNAL MAPPING LAYER ───────────────────────────────────────────────────

func (h *PaymentGRPCHandler) toProtoPayment(p *entity.Payment) *paymentpb.Payment {
	if p == nil {
		return nil
	}

	res := &paymentpb.Payment{
		Id:            p.ID,
		OrderId:       p.OrderID,
		TransactionId: p.TransactionID,
		Amount:        p.Amount,
		CreatedAt:     timestamppb.New(p.CreatedAt),
		UpdatedAt:     timestamppb.New(p.UpdatedAt),
	}

	switch p.Method {
	case entity.PaymentMethodCOD:
		res.Method = paymentpb.PaymentMethod_PAYMENT_METHOD_COD
	case entity.PaymentMethodOnline:
		res.Method = paymentpb.PaymentMethod_PAYMENT_METHOD_ONLINE
	}

	switch p.Status {
	case entity.PaymentStatusPending:
		res.Status = paymentpb.PaymentStatus_PAYMENT_STATUS_PENDING
	case entity.PaymentStatusCompleted:
		res.Status = paymentpb.PaymentStatus_PAYMENT_STATUS_COMPLETED
	case entity.PaymentStatusFailed:
		res.Status = paymentpb.PaymentStatus_PAYMENT_STATUS_FAILED
	case entity.PaymentStatusRefunded:
		res.Status = paymentpb.PaymentStatus_PAYMENT_STATUS_REFUNDED
	}

	if p.OnlineTransaction != nil {
		var prov paymentpb.PaymentProvider
		switch p.OnlineTransaction.Provider {
		case entity.PaymentProviderStripe:
			prov = paymentpb.PaymentProvider_PAYMENT_PROVIDER_STRIPE
		case entity.PaymentProviderSSLCOMMERZ:
			prov = paymentpb.PaymentProvider_PAYMENT_PROVIDER_SSLCOMMERZ
		case entity.PaymentProviderPaypal:
			prov = paymentpb.PaymentProvider_PAYMENT_PROVIDER_PAYPAL
		}

		res.Details = &paymentpb.Payment_OnlineTransaction{
			OnlineTransaction: &paymentpb.OnlineTransaction{
				Id:            p.OnlineTransaction.ID,
				PaymentId:     p.OnlineTransaction.PaymentID,
				Provider:      prov,
				GatewayRef:    p.OnlineTransaction.GatewayRef,
				GatewayStatus: p.OnlineTransaction.GatewayStatus,
				RawResponse:   string(p.OnlineTransaction.RawResponse),
				CreatedAt:     timestamppb.New(p.OnlineTransaction.CreatedAt),
			},
		}
	} else if p.CODDetail != nil {
		cod := &paymentpb.CODDetail{
			Id:        p.CODDetail.ID,
			PaymentId: p.CODDetail.PaymentID,
			CreatedAt: timestamppb.New(p.CODDetail.CreatedAt),
			UpdatedAt: timestamppb.New(p.CODDetail.UpdatedAt),
		}
		if p.CODDetail.CollectedAt != nil {
			cod.CollectedAt = timestamppb.New(*p.CODDetail.CollectedAt)
		}
		res.Details = &paymentpb.Payment_CodDetail{
			CodDetail: cod,
		}
	}

	return res
}
