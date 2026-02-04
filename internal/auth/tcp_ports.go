package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type ReservedTCPPort struct {
	Port    int
	TokenID int64
	// TokenPrefix is a short display-safe token prefix.
	TokenPrefix string

	CreatedAt time.Time
}

func (s *Store) ReserveTCPPort(ctx context.Context, tokenID int64, port int) error {
	if s == nil || s.db == nil {
		return errors.New("nil store")
	}
	if tokenID <= 0 {
		return errors.New("invalid token id")
	}
	if port <= 0 || port > 65535 {
		return errors.New("invalid port")
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO reserved_tcp_ports (port, token_id, created_at)
		VALUES (?, ?, ?)
	`, port, tokenID, time.Now().UTC().Unix())
	return err
}

func (s *Store) UnreserveTCPPort(ctx context.Context, port int) error {
	if s == nil || s.db == nil {
		return errors.New("nil store")
	}
	if port <= 0 || port > 65535 {
		return errors.New("invalid port")
	}

	_, err := s.db.ExecContext(ctx, `
		DELETE FROM reserved_tcp_ports
		WHERE port = ?
	`, port)
	return err
}

func (s *Store) ListReservedTCPPorts(ctx context.Context) ([]ReservedTCPPort, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("nil store")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT r.port, r.token_id, t.token_prefix, r.created_at
		FROM reserved_tcp_ports r
		JOIN authtokens t ON t.id = r.token_id
		ORDER BY r.port ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ReservedTCPPort
	for rows.Next() {
		var (
			rec       ReservedTCPPort
			createdAt int64
		)
		if err := rows.Scan(&rec.Port, &rec.TokenID, &rec.TokenPrefix, &createdAt); err != nil {
			return nil, err
		}
		rec.CreatedAt = time.Unix(createdAt, 0).UTC()
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Store) ReservedTCPPortTokenID(ctx context.Context, port int) (int64, bool, error) {
	if s == nil || s.db == nil {
		return 0, false, errors.New("nil store")
	}
	if port <= 0 || port > 65535 {
		return 0, false, errors.New("invalid port")
	}

	var tokenID int64
	err := s.db.QueryRowContext(ctx, `
		SELECT token_id
		FROM reserved_tcp_ports
		WHERE port = ?
		LIMIT 1
	`, port).Scan(&tokenID)
	if err == nil {
		return tokenID, true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return 0, false, err
}
