package middleware

import (
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

var rateLimitClients sync.Map

func cleanupStaleRateLimiters() {
	now := time.Now()
	rateLimitClients.Range(func(key, value any) bool {
		entry := value.(*rateLimitClientEntry)
		if now.Sub(entry.lastSeen) > 5*time.Minute {
			rateLimitClients.Delete(key)
		}
		return true
	})
}

func init() {
	go func() {
		for {
			time.Sleep(3 * time.Minute)
			cleanupStaleRateLimiters()
		}
	}()
}

func getRateLimitLimiter(ip string) *rate.Limiter {
	if val, ok := rateLimitClients.Load(ip); ok {
		entry := val.(*rateLimitClientEntry)
		entry.lastSeen = time.Now()
		return entry.limiter
	}

	limiter := rate.NewLimiter(rate.Every(6*time.Second), 10)
	rateLimitClients.Store(ip, &rateLimitClientEntry{
		limiter:  limiter,
		lastSeen: time.Now(),
	})
	return limiter
}

func RateLimitClaims(c *gin.Context) {
	if c.Request.Method == "OPTIONS" {
		c.Next()
		return
	}

	ip := c.ClientIP()
	limiter := getRateLimitLimiter(ip)

	if !limiter.Allow() {
		c.Header("Retry-After", "6")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
		c.Abort()
		return
	}

	c.Next()
}

// shareCardRateLimitClients is a separate rate limiter for the share card PNG endpoint.
var shareCardRateLimitClients sync.Map

func cleanupStaleShareCardRateLimiters() {
	now := time.Now()
	shareCardRateLimitClients.Range(func(key, value any) bool {
		entry := value.(*rateLimitClientEntry)
		if now.Sub(entry.lastSeen) > 5*time.Minute {
			shareCardRateLimitClients.Delete(key)
		}
		return true
	})
}

func getShareCardRateLimitLimiter(ip string) *rate.Limiter {
	if val, ok := shareCardRateLimitClients.Load(ip); ok {
		entry := val.(*rateLimitClientEntry)
		entry.lastSeen = time.Now()
		return entry.limiter
	}

	// 10 requests per minute (burst of 10, refill 1 every 6s)
	limiter := rate.NewLimiter(rate.Every(6*time.Second), 10)
	shareCardRateLimitClients.Store(ip, &rateLimitClientEntry{
		limiter:  limiter,
		lastSeen: time.Now(),
	})
	return limiter
}

func init() {
	// Start cleanup for share card rate limiters
	go func() {
		for {
			time.Sleep(3 * time.Minute)
			cleanupStaleShareCardRateLimiters()
		}
	}()
}

// RateLimitShareCard limits the share card PNG endpoint to 10 requests per IP per minute.
func RateLimitShareCard(c *gin.Context) {
	ip := c.ClientIP()
	limiter := getShareCardRateLimitLimiter(ip)

	if !limiter.Allow() {
		c.Header("Retry-After", "6")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
		c.Abort()
		return
	}

	c.Next()
}
