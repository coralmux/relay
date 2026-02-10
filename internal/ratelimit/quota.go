package ratelimit

import "github.com/openclaw/openclaw-relay/internal/store"

const (
	DailyQuotaBytes   int64 = 500 * 1024 * 1024       // 500 MB
	MonthlyQuotaBytes int64 = 10 * 1024 * 1024 * 1024  // 10 GB
)

// QuotaChecker checks bandwidth quotas against the store.
type QuotaChecker struct {
	store store.Store
}

func NewQuotaChecker(s store.Store) *QuotaChecker {
	return &QuotaChecker{store: s}
}

// Check returns an error code string if quota is exceeded, or empty string if within quota.
func (q *QuotaChecker) Check(token string) (string, error) {
	daily, err := q.store.GetDailyUsage(token)
	if err != nil {
		return "", err
	}
	if daily >= DailyQuotaBytes {
		return "DAILY_QUOTA_EXCEEDED", nil
	}

	monthly, err := q.store.GetMonthlyUsage(token)
	if err != nil {
		return "", err
	}
	if monthly >= MonthlyQuotaBytes {
		return "MONTHLY_QUOTA_EXCEEDED", nil
	}

	return "", nil
}

// Record records bytes used for a token.
func (q *QuotaChecker) Record(token string, bytes int64) error {
	return q.store.RecordBytes(token, bytes)
}
