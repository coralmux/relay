package store

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS tokens (
			token TEXT PRIMARY KEY,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS bandwidth (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token TEXT NOT NULL,
			bytes INTEGER NOT NULL,
			recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (token) REFERENCES tokens(token) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bandwidth_token_date ON bandwidth(token, recorded_at)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) CreateToken(token string) error {
	_, err := s.db.Exec("INSERT OR IGNORE INTO tokens (token) VALUES (?)", token)
	return err
}

func (s *SQLiteStore) DeleteToken(token string) error {
	_, err := s.db.Exec("DELETE FROM tokens WHERE token = ?", token)
	return err
}

func (s *SQLiteStore) TokenExists(token string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM tokens WHERE token = ?", token).Scan(&count)
	return count > 0, err
}

func (s *SQLiteStore) ListTokens() ([]TokenInfo, error) {
	rows, err := s.db.Query("SELECT token, created_at FROM tokens ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []TokenInfo
	for rows.Next() {
		var t TokenInfo
		if err := rows.Scan(&t.Token, &t.CreatedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func (s *SQLiteStore) RecordBytes(token string, bytes int64) error {
	_, err := s.db.Exec("INSERT INTO bandwidth (token, bytes) VALUES (?, ?)", token, bytes)
	return err
}

func (s *SQLiteStore) GetDailyUsage(token string) (int64, error) {
	today := time.Now().Format("2006-01-02")
	var total sql.NullInt64
	err := s.db.QueryRow(
		"SELECT SUM(bytes) FROM bandwidth WHERE token = ? AND DATE(recorded_at) = ?",
		token, today,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	if total.Valid {
		return total.Int64, nil
	}
	return 0, nil
}

func (s *SQLiteStore) GetMonthlyUsage(token string) (int64, error) {
	monthStart := time.Now().Format("2006-01") + "-01"
	var total sql.NullInt64
	err := s.db.QueryRow(
		"SELECT SUM(bytes) FROM bandwidth WHERE token = ? AND DATE(recorded_at) >= ?",
		token, monthStart,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	if total.Valid {
		return total.Int64, nil
	}
	return 0, nil
}

func (s *SQLiteStore) ResetDailyUsage() error {
	yesterday := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	_, err := s.db.Exec("DELETE FROM bandwidth WHERE DATE(recorded_at) < ?", yesterday)
	return err
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
