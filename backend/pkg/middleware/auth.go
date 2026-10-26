package middleware

import (
	"booking-system/pkg/auth"
	"booking-system/pkg/response"

	"github.com/gin-gonic/gin"
)

// Auth builds middleware that rejects requests with a 401 unless the access_token cookie is valid.
// Takes nothing; returns a gin.HandlerFunc that stores the token's *auth.UserInfo under "user" for later handlers.
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
