//go:build integration

package integration

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestServerHealthz(t *testing.T) {
	t.Parallel()

	baseURL := strings.TrimRight(getenv("EOSRIFT_SERVER_URL", "http://server:8080"), "/")

	deadline := time.Now().Add(10 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/healthz")
		if err != nil {
			lastErr = err
			time.Sleep(200 * time.Millisecond)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = &unexpectedStatusError{statusCode: resp.StatusCode, body: string(body)}
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if got := string(body); got != "ok\n" {
			t.Fatalf("body = %q, want %q", got, "ok\\n")
		}

		return
	}

	if lastErr == nil {
		t.Fatalf("health check did not succeed before deadline")
	}
	t.Fatalf("health check did not succeed before deadline: %v", lastErr)
}

type unexpectedStatusError struct {
	statusCode int
	body       string
}

func (e *unexpectedStatusError) Error() string {
	return "unexpected status: " + http.StatusText(e.statusCode)
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

