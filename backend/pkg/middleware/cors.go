package middleware

import (
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS allows exactly one origin: the whole value of ALLOWED_ORIGINS. It isn't split on commas.
func CORS() gin.HandlerFunc {
	var ev = os.Getenv("ALLOWED_ORIGINS")
	return cors.New(cors.Config{
		AllowOrigins:     []string{ev},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
