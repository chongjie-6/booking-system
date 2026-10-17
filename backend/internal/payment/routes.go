package payment

import (
	"github.com/gin-gonic/gin"
)

func RegisterPaymentRoutes(r *gin.Engine){
	paymentRoutes := r.Group("/payments")
	{
		paymentRoutes.GET("/ping")
	}
}