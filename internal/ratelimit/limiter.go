package ratelimit

import (
	"golang.org/x/time/rate"
)

const (
	PhoneMessagesPerMinute = 30
	AgentMessagesPerMinute = 120
	MaxMessageSize         = 5 * 1024 * 1024 // 5MB
)

// NewPhoneLimiter creates a rate limiter for phone connections.
func NewPhoneLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Limit(float64(PhoneMessagesPerMinute)/60.0), PhoneMessagesPerMinute)
}

// NewAgentLimiter creates a rate limiter for agent connections.
func NewAgentLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Limit(float64(AgentMessagesPerMinute)/60.0), AgentMessagesPerMinute)
}
