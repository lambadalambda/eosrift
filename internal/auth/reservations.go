package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

func (s *Store) ReserveSubdomain(ctx context.Context, tokenID int64, subdomain string) error {
	if s == nil || s.db == nil {
		return errors.New("nil store")
	}
	if tokenID <= 0 {
		return errors.New("invalid token id")
	}

	norm, err := normalizeSubdomain(subdomain)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO reserved_subdomains (subdomain, token_id, created_at)
		VALUES (?, ?, ?)
	`, norm, tokenID, time.Now().UTC().Unix())
	return err
}

func (s *Store) UnreserveSubdomain(ctx context.Context, subdomain string) error {
	if s == nil || s.db == nil {
		return errors.New("nil store")
	}

	norm, err := normalizeSubdomain(subdomain)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		DELETE FROM reserved_subdomains
		WHERE subdomain = ?
	`, norm)
	return err
}

func (s *Store) ReservedSubdomainTokenID(ctx context.Context, subdomain string) (int64, bool, error) {
	if s == nil || s.db == nil {
		return 0, false, errors.New("nil store")
	}

	norm, err := normalizeSubdomain(subdomain)
	if err != nil {
		return 0, false, err
	}

	var tokenID int64
	err = s.db.QueryRowContext(ctx, `
		SELECT token_id
		FROM reserved_subdomains
		WHERE subdomain = ?
		LIMIT 1
	`, norm).Scan(&tokenID)
	if err == nil {
		return tokenID, true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return 0, false, err
}

func normalizeSubdomain(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", errors.New("empty subdomain")
	}
	if strings.Contains(s, ".") {
		return "", errors.New("invalid subdomain")
	}
	if len(s) > 63 {
		return "", errors.New("subdomain too long")
	}
	if s[0] == '-' || s[len(s)-1] == '-' {
		return "", errors.New("invalid subdomain")
	}

	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			continue
		}
		if c >= '0' && c <= '9' {
			continue
		}
		if c == '-' {
			continue
		}
		return "", errors.New("invalid subdomain")
	}

	return s, nil
}

