//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestAdminAPI_TokensRequiresAuth(t *testing.T) {
	t.Parallel()

	clientHTTP := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(http.MethodGet, httpURL("/api/admin/tokens"), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = "eosrift.test"

	resp, err := clientHTTP.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestAdminAPI_TokensList(t *testing.T) {
	t.Parallel()

	clientHTTP := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest(http.MethodGet, httpURL("/api/admin/tokens"), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = "eosrift.test"
	req.Header.Set("Authorization", "Bearer "+getenv("EOSRIFT_ADMIN_TOKEN", ""))

	resp, err := clientHTTP.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body struct {
		Tokens []struct {
			ID int64 `json:"id"`
		} `json:"tokens"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Tokens) == 0 {
		t.Fatalf("tokens len = 0, want >= 1")
	}
}

