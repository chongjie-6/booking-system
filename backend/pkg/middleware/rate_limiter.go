package middleware

import (
	"booking-system/pkg/response"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func RateLimiter() gin.HandlerFunc {
	type client struct {
		limiter *rate.Limiter
	}

	const (
		REQ_PER_SEC       = 10
		BURST_REQ_PER_SEC = 20
	)

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		mu.Lock()

		if _, exists := clients[ip]; !exists {
			clients[ip] = &client{limiter: rate.NewLimiter(REQ_PER_SEC, BURST_REQ_PER_SEC)}
		}

		cl := clients[ip]
		mu.Unlock()

		if !cl.limiter.Allow() {
			c.Error(response.ErrTooManyRequests)
			c.Abort()
		}

		c.Next()
	}

}
