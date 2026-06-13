package payment

import (
	"github.com/gin-gonic/gin"
	"github.com/mafi020/ecom-golang-micro/config"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/http/handler"
	"github.com/mafi020/ecom-golang-micro/internal/delivery/http/middleware"
)

func RegisterHTTPHandlers(r *gin.Engine, uc *Usecases, cfg *config.Config) {
	paymentHandler := handler.NewPaymentHandler(uc.PaymentUC)

	// Gateway strips /api/payments, service receives /online, /cod, /order/:order_id
	payment := r.Group("")
	payment.Use(middleware.AuthMiddleware())
	{
		payment.POST("/online", paymentHandler.PayOnline)
		payment.POST("/cod", paymentHandler.PayCOD)
		payment.GET("/order/:order_id", paymentHandler.GetPaymentByOrderID)
	}
}
