package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Token struct {
	ID int64

	Label  string
	Prefix string

	CreatedAt time.Time
	RevokedAt *time.Time
}

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, dbPath string) (*Store, error) {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		return nil, errors.New("empty db path")
	}

	dsn := dbPath
	if !strings.HasPrefix(dsn, "file:") && dsn != ":memory:" {
		dsn = "file:" + dsn
	}

	// Pragmas:
	// - busy_timeout: friendlier under contention.
	// - journal_mode=WAL: better concurrency; safe for most deployments.
	// - foreign_keys=ON: future-proof.
	if strings.Contains(dsn, "?") {
		dsn += "&"
	} else {
		dsn += "?"
	}
	dsn += "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Keep it simple/portable for now.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	s := &Store{db: db}

	if err := s.db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := s.ensureSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) CreateToken(ctx context.Context, label string) (Token, string, error) {
	var rec Token

	if s == nil || s.db == nil {
		return rec, "", errors.New("nil store")
	}

	plain, err := generateToken()
	if err != nil {
		return rec, "", err
	}

	rec, err = s.insertToken(ctx, plain, label)
	if err != nil {
		return rec, "", err
	}

	return rec, plain, nil
}

func (s *Store) EnsureToken(ctx context.Context, token, label string) error {
	if s == nil || s.db == nil {
		return errors.New("nil store")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("empty token")
	}

	if _, err := s.insertToken(ctx, token, label); err != nil {
		// If it already exists, treat as success.
		if isUniqueViolation(err) {
			return nil
		}
		return err
	}

	return nil
}

func (s *Store) ListTokens(ctx context.Context) ([]Token, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("nil store")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, label, token_prefix, created_at, revoked_at
		FROM authtokens
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Token
	for rows.Next() {
		var (
			rec       Token
			createdAt int64
			revokedAt sql.NullInt64
		)

		if err := rows.Scan(&rec.ID, &rec.Label, &rec.Prefix, &createdAt, &revokedAt); err != nil {
			return nil, err
		}

		rec.CreatedAt = time.Unix(createdAt, 0).UTC()
		if revokedAt.Valid {
			ts := time.Unix(revokedAt.Int64, 0).UTC()
			rec.RevokedAt = &ts
		}

		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Store) RevokeToken(ctx context.Context, id int64) error {
	if s == nil || s.db == nil {
		return errors.New("nil store")
	}
	if id <= 0 {
		return errors.New("invalid token id")
	}

	now := time.Now().UTC().Unix()
	res, err := s.db.ExecContext(ctx, `
		UPDATE authtokens
		SET revoked_at = ?
		WHERE id = ? AND revoked_at IS NULL
	`, now, id)
	if err != nil {
		return err
	}
	_, _ = res.RowsAffected()
	return nil
}

func (s *Store) ValidateToken(ctx context.Context, token string) (bool, error) {
	if s == nil || s.db == nil {
		return false, errors.New("nil store")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return false, nil
	}

	hash := hashToken(token)

	var exists int
	err := s.db.QueryRowContext(ctx, `
		SELECT 1
		FROM authtokens
		WHERE token_hash = ? AND revoked_at IS NULL
		LIMIT 1
	`, hash).Scan(&exists)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func (s *Store) TokenID(ctx context.Context, token string) (int64, bool, error) {
	if s == nil || s.db == nil {
		return 0, false, errors.New("nil store")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return 0, false, nil
	}

	hash := hashToken(token)

	var id int64
	err := s.db.QueryRowContext(ctx, `
		SELECT id
		FROM authtokens
		WHERE token_hash = ? AND revoked_at IS NULL
		LIMIT 1
	`, hash).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return 0, false, err
}

func (s *Store) ensureSchema(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("nil store")
	}

	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS authtokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			label TEXT NOT NULL DEFAULT '',
			token_hash TEXT NOT NULL UNIQUE,
			token_prefix TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			revoked_at INTEGER
		);
	`); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS authtokens_revoked_at ON authtokens(revoked_at);
	`); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS reserved_subdomains (
			subdomain TEXT PRIMARY KEY,
			token_id INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			FOREIGN KEY(token_id) REFERENCES authtokens(id) ON DELETE CASCADE
		);
	`); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS reserved_subdomains_token_id ON reserved_subdomains(token_id);
	`); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS reserved_tcp_ports (
			port INTEGER PRIMARY KEY,
			token_id INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			FOREIGN KEY(token_id) REFERENCES authtokens(id) ON DELETE CASCADE
		);
	`); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS reserved_tcp_ports_token_id ON reserved_tcp_ports(token_id);
	`); err != nil {
		return err
	}

	return nil
}

func (s *Store) insertToken(ctx context.Context, token, label string) (Token, error) {
	var rec Token

	hash := hashToken(token)
	prefix := tokenPrefix(token)
	createdAt := time.Now().UTC()

	res, err := s.db.ExecContext(ctx, `
		INSERT INTO authtokens (label, token_hash, token_prefix, created_at)
		VALUES (?, ?, ?, ?)
	`, strings.TrimSpace(label), hash, prefix, createdAt.Unix())
	if err != nil {
		return rec, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return rec, err
	}

	rec = Token{
		ID:        id,
		Label:     strings.TrimSpace(label),
		Prefix:    prefix,
		CreatedAt: createdAt,
	}

	return rec, nil
}

func tokenPrefix(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 12 {
		return token
	}
	return token[:12]
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func generateToken() (string, error) {
	// 32 random bytes ~ 52 base32 chars.
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	s := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
	s = strings.ToLower(s)
	return "eos_" + s, nil
}

func isUniqueViolation(err error) bool {
	// modernc sqlite returns an error string like:
	// "constraint failed: UNIQUE constraint failed: ..."
	// Avoid importing driver-specific error types.
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}
