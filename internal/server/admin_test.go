package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"eosrift.com/eosrift/internal/auth"
)

func TestNewHandler_AdminUI_BaseDomainOnly(t *testing.T) {
	t.Parallel()

	h := NewHandler(Config{
		BaseDomain:   "eosrift.com",
		TunnelDomain: "tunnel.eosrift.com",
		AdminToken:   "admin-secret",
	}, Dependencies{
		AdminStore: newStubAdminStore(),
	})

	req := httptest.NewRequest(http.MethodGet, "http://eosrift.com/admin", nil)
	req.Host = "eosrift.com"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "Eosrift Admin") {
		t.Fatalf("missing admin page marker")
	}

	req2 := httptest.NewRequest(http.MethodGet, "http://abcd1234.tunnel.eosrift.com/admin", nil)
	req2.Host = "abcd1234.tunnel.eosrift.com"
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec2.Code, http.StatusNotFound)
	}
}

func TestNewHandler_AdminAPI_Auth(t *testing.T) {
	t.Parallel()

	store := newStubAdminStore()
	store.tokens = []auth.Token{
		{ID: 1, Prefix: "eos_abc", Label: "laptop", CreatedAt: time.Unix(1700000000, 0).UTC()},
	}

	h := NewHandler(Config{
		BaseDomain:   "eosrift.com",
		TunnelDomain: "tunnel.eosrift.com",
		AdminToken:   "admin-secret",
	}, Dependencies{
		AdminStore: store,
	})

	makeReq := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "http://eosrift.com/api/admin/tokens", nil)
		req.Host = "eosrift.com"
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	if got := makeReq(""); got.Code != http.StatusUnauthorized {
		t.Fatalf("status(no auth) = %d, want %d", got.Code, http.StatusUnauthorized)
	}
	if got := makeReq("wrong"); got.Code != http.StatusUnauthorized {
		t.Fatalf("status(wrong auth) = %d, want %d", got.Code, http.StatusUnauthorized)
	}

	ok := makeReq("admin-secret")
	if ok.Code != http.StatusOK {
		t.Fatalf("status(ok auth) = %d, want %d (body=%q)", ok.Code, http.StatusOK, ok.Body.String())
	}
	if !strings.Contains(ok.Body.String(), `"tokens"`) {
		t.Fatalf("body missing tokens list: %q", ok.Body.String())
	}
}

func TestNewHandler_AdminAPI_Mutations(t *testing.T) {
	t.Parallel()

	store := newStubAdminStore()
	h := NewHandler(Config{
		BaseDomain:   "eosrift.com",
		TunnelDomain: "tunnel.eosrift.com",
		AdminToken:   "admin-secret",
	}, Dependencies{
		AdminStore: store,
	})

	doJSON := func(method, path string, body any) *httptest.ResponseRecorder {
		var payload []byte
		if body != nil {
			payload, _ = json.Marshal(body)
		}
		req := httptest.NewRequest(method, "http://eosrift.com"+path, bytes.NewReader(payload))
		req.Host = "eosrift.com"
		req.Header.Set("Authorization", "Bearer admin-secret")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	createToken := doJSON(http.MethodPost, "/api/admin/tokens", map[string]any{"label": "runner"})
	if createToken.Code != http.StatusCreated {
		t.Fatalf("create token status = %d, want %d (body=%q)", createToken.Code, http.StatusCreated, createToken.Body.String())
	}
	if !strings.Contains(createToken.Body.String(), `"token"`) {
		t.Fatalf("create token missing plain token: %q", createToken.Body.String())
	}

	resSubdomain := doJSON(http.MethodPost, "/api/admin/subdomains", map[string]any{"token_id": 1, "subdomain": "demo"})
	if resSubdomain.Code != http.StatusCreated {
		t.Fatalf("reserve subdomain status = %d, want %d (body=%q)", resSubdomain.Code, http.StatusCreated, resSubdomain.Body.String())
	}

	resPort := doJSON(http.MethodPost, "/api/admin/tcp-ports", map[string]any{"token_id": 1, "port": 20005})
	if resPort.Code != http.StatusCreated {
		t.Fatalf("reserve port status = %d, want %d (body=%q)", resPort.Code, http.StatusCreated, resPort.Body.String())
	}

	revoke := doJSON(http.MethodDelete, "/api/admin/tokens/1", nil)
	if revoke.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d, want %d (body=%q)", revoke.Code, http.StatusNoContent, revoke.Body.String())
	}

	delSub := doJSON(http.MethodDelete, "/api/admin/subdomains/demo", nil)
	if delSub.Code != http.StatusNoContent {
		t.Fatalf("delete subdomain status = %d, want %d (body=%q)", delSub.Code, http.StatusNoContent, delSub.Body.String())
	}

	delPort := doJSON(http.MethodDelete, "/api/admin/tcp-ports/20005", nil)
	if delPort.Code != http.StatusNoContent {
		t.Fatalf("delete port status = %d, want %d (body=%q)", delPort.Code, http.StatusNoContent, delPort.Body.String())
	}
}

type stubAdminStore struct {
	mu sync.Mutex

	nextTokenID int64

	tokens     []auth.Token
	subdomains []auth.ReservedSubdomain
	ports      []auth.ReservedTCPPort
}

func newStubAdminStore() *stubAdminStore {
	return &stubAdminStore{
		nextTokenID: 1,
	}
}

func (s *stubAdminStore) CreateToken(ctx context.Context, label string) (auth.Token, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextTokenID
	s.nextTokenID++
	rec := auth.Token{
		ID:        id,
		Label:     label,
		Prefix:    "eos_prefix_" + strconv.FormatInt(id, 10),
		CreatedAt: time.Unix(1700000000+id, 0).UTC(),
	}
	s.tokens = append(s.tokens, rec)
	return rec, "eos_token_" + strconv.FormatInt(id, 10), nil
}

func (s *stubAdminStore) ListTokens(ctx context.Context) ([]auth.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]auth.Token, len(s.tokens))
	copy(out, s.tokens)
	return out, nil
}

func (s *stubAdminStore) RevokeToken(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for i := range s.tokens {
		if s.tokens[i].ID == id {
			s.tokens[i].RevokedAt = &now
			return nil
		}
	}
	return nil
}

func (s *stubAdminStore) ListReservedSubdomains(ctx context.Context) ([]auth.ReservedSubdomain, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]auth.ReservedSubdomain, len(s.subdomains))
	copy(out, s.subdomains)
	return out, nil
}

func (s *stubAdminStore) ReserveSubdomain(ctx context.Context, tokenID int64, subdomain string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subdomains = append(s.subdomains, auth.ReservedSubdomain{
		Subdomain:   subdomain,
		TokenID:     tokenID,
		TokenPrefix: "eos_prefix_" + strconv.FormatInt(tokenID, 10),
		CreatedAt:   time.Now().UTC(),
	})
	return nil
}

func (s *stubAdminStore) UnreserveSubdomain(ctx context.Context, subdomain string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	keep := s.subdomains[:0]
	for _, rec := range s.subdomains {
		if rec.Subdomain == subdomain {
			continue
		}
		keep = append(keep, rec)
	}
	s.subdomains = keep
	return nil
}

func (s *stubAdminStore) ListReservedTCPPorts(ctx context.Context) ([]auth.ReservedTCPPort, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]auth.ReservedTCPPort, len(s.ports))
	copy(out, s.ports)
	return out, nil
}

func (s *stubAdminStore) ReserveTCPPort(ctx context.Context, tokenID int64, port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ports = append(s.ports, auth.ReservedTCPPort{
		Port:        port,
		TokenID:     tokenID,
		TokenPrefix: "eos_prefix_" + strconv.FormatInt(tokenID, 10),
		CreatedAt:   time.Now().UTC(),
	})
	return nil
}

func (s *stubAdminStore) UnreserveTCPPort(ctx context.Context, port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	keep := s.ports[:0]
	for _, rec := range s.ports {
		if rec.Port == port {
			continue
		}
		keep = append(keep, rec)
	}
	s.ports = keep
	return nil
}
