package store

import "time"

// Store abstracts persistence for pairing tokens and quota tracking.
type Store interface {
	// Token management
	CreateToken(token string) error
	DeleteToken(token string) error
	TokenExists(token string) (bool, error)
	ListTokens() ([]TokenInfo, error)

	// Quota tracking
	RecordBytes(token string, bytes int64) error
	GetDailyUsage(token string) (int64, error)
	GetMonthlyUsage(token string) (int64, error)
	ResetDailyUsage() error

	Close() error
}

type TokenInfo struct {
	Token     string
	CreatedAt time.Time
}
