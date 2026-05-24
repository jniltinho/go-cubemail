package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
	"golang.org/x/time/rate"
)

// ipLimiter associates a token-bucket rate limiter with a timestamp for cleanup.
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	loginMu       sync.Mutex
	loginLimiters = make(map[string]*ipLimiter)
)

func init() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			loginMu.Lock()
			for ip, il := range loginLimiters {
				if time.Since(il.lastSeen) > 5*time.Minute {
					delete(loginLimiters, ip)
				}
			}
			loginMu.Unlock()
		}
	}()
}

// LoginRateLimit allows 3 burst attempts then 1 per 12 seconds (5/min) per IP.
func LoginRateLimit() echo.MiddlewareFunc {
	return NewRateLimit(5, time.Minute)
}

// NewRateLimit creates a per-IP rate limiter with the given rate and window.
// maxRequests is the maximum number of requests allowed per window.
func NewRateLimit(maxRequests int, window time.Duration) echo.MiddlewareFunc {
	interval := window / time.Duration(maxRequests)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			ip := c.RealIP()

			loginMu.Lock()
			il, ok := loginLimiters[ip]
			if !ok {
				il = &ipLimiter{limiter: rate.NewLimiter(rate.Every(interval), maxRequests)}
				loginLimiters[ip] = il
			}
			il.lastSeen = time.Now()
			allowed := il.limiter.Allow()
			loginMu.Unlock()

			if !allowed {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "Too many requests. Please try again later.",
				})
			}
			return next(c)
		}
	}
}
