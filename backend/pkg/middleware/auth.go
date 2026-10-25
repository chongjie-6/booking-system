package middleware

import (
	"booking-system/pkg/auth"
	"booking-system/pkg/response"

	"github.com/gin-gonic/gin"
)

// Auth rejects requests without a valid cookie
// and stores the token's auth.UserInfo under "user" for later handlers.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("access_token")
		if err != nil {
			c.Error(response.ErrUnauthorized)
			c.Abort()
			return
		}

		claims, err := auth.ParseJWT(token)
		if err != nil || claims.User == nil {
			c.Error(response.ErrUnauthorized)
			c.Abort()
			return
		}

		c.Set("user", claims.User)
		c.Next()
	}
}
