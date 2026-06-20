package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type rateLimitClientEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipRateLimiter struct {
	clients sync.Map
	every   time.Duration
	burst   int
}

func newIPRateLimiter(every time.Duration, burst int) *ipRateLimiter {
	rl := &ipRateLimiter{every: every, burst: burst}
	go func() {
		for {
			time.Sleep(3 * time.Minute)
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *ipRateLimiter) cleanup() {
	now := time.Now()
	rl.clients.Range(func(key, value any) bool {
		if now.Sub(value.(*rateLimitClientEntry).lastSeen) > 5*time.Minute {
			rl.clients.Delete(key)
		}
		return true
	})
}

func (rl *ipRateLimiter) allow(ip string) bool {
	if val, ok := rl.clients.Load(ip); ok {
		entry := val.(*rateLimitClientEntry)
		entry.lastSeen = time.Now()
		return entry.limiter.Allow()
	}
	limiter := rate.NewLimiter(rate.Every(rl.every), rl.burst)
	rl.clients.Store(ip, &rateLimitClientEntry{limiter: limiter, lastSeen: time.Now()})
	return limiter.Allow()
}

func (rl *ipRateLimiter) middleware() gin.HandlerFunc {
	retryAfter := fmt.Sprintf("%d", int(rl.every.Seconds()))
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			c.Header("Retry-After", retryAfter)
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}

var (
	claimsLimiter    = newIPRateLimiter(6*time.Second, 10)
	shareCardLimiter = newIPRateLimiter(6*time.Second, 10)
	buyerLimiter     = newIPRateLimiter(2*time.Second, 20)
)

// RateLimitClaims limits spot claim submissions to 10 per IP per minute.
func RateLimitClaims(c *gin.Context) {
	if c.Request.Method == "OPTIONS" {
		c.Next()
		return
	}
	claimsLimiter.middleware()(c)
}

// RateLimitShareCard limits the share card PNG endpoint to 10 requests per IP per minute.
func RateLimitShareCard(c *gin.Context) {
	shareCardLimiter.middleware()(c)
}

// RateLimitBuyer limits public buyer lookup endpoints to 20 per IP per 2 seconds (burst)
// to prevent handle enumeration while staying invisible to normal browsing.
func RateLimitBuyer(c *gin.Context) {
	buyerLimiter.middleware()(c)
}
