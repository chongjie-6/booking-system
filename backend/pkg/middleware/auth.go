package middleware

import (
	"booking-system/pkg/auth"
	"booking-system/pkg/response"

	"github.com/gin-gonic/gin"
)

// Auth rejects requests without a valid "Authorization: Bearer <jwt>" header
// and stores the token's auth.UserInfo under "user" for later handlers.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("auth_token")
		if err != nil {
			c.Error(response.ErrUnauthorized)
			c.Abort()
			return
		}

		claims, err := auth.ParseJWT(token)
		if err != nil {
			c.Error(response.ErrUnauthorized)
			c.Abort()
			return
		}

		c.Set("user", claims.Sub)
		c.Next()
	}
}
