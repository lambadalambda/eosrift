package deployhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHandler_RejectsInvalidSignature(t *testing.T) {
	t.Parallel()

	var called atomic.Int32
	h, err := NewHandler(Config{
		Secret:       "secret",
		WorkflowName: "Docker Image",
		Branch:       "main",
	}, func(context.Context, Run) error {
		called.Add(1)
		return nil
	}, log.New(ioDiscard{}, "", 0))
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	payload := []byte(`{"action":"completed"}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/deploy", strings.NewReader(string(payload)))
	req.Header.Set("X-GitHub-Event", "workflow_run")
	req.Header.Set("X-Hub-Signature-256", "sha256=deadbeef")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if called.Load() != 0 {
		t.Fatalf("trigger called %d times, want 0", called.Load())
	}
}

func TestHandler_IgnoresNonMatchingWorkflow(t *testing.T) {
	t.Parallel()

	var called atomic.Int32
	h := mustHandler(t, Config{
		Secret:       "secret",
		WorkflowName: "Docker Image",
		Branch:       "main",
		Repository:   "lambadalambda/eosrift",
	}, func(context.Context, Run) error {
		called.Add(1)
		return nil
	})

	payload := []byte(`{
		"action":"completed",
		"repository":{"full_name":"lambadalambda/eosrift"},
		"workflow_run":{
			"id":123,
			"name":"CI",
			"conclusion":"success",
			"head_branch":"main",
			"head_sha":"abc123",
			"html_url":"https://github.com/example"
		}
	}`)

	rec := doSignedRequest(t, h, payload, "secret", "workflow_run")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if called.Load() != 0 {
		t.Fatalf("trigger called %d times, want 0", called.Load())
	}
}

func TestHandler_TriggersOnMatchingSuccessfulWorkflow(t *testing.T) {
	t.Parallel()

	triggerCh := make(chan Run, 1)
	h := mustHandler(t, Config{
		Secret:       "secret",
		WorkflowName: "Docker Image",
		Branch:       "main",
		Repository:   "lambadalambda/eosrift",
	}, func(_ context.Context, run Run) error {
		triggerCh <- run
		return nil
	})

	payload := []byte(`{
		"action":"completed",
		"repository":{"full_name":"lambadalambda/eosrift"},
		"workflow_run":{
			"id":456,
			"name":"Docker Image",
			"conclusion":"success",
			"head_branch":"main",
			"head_sha":"deadcafe",
			"html_url":"https://github.com/lambadalambda/eosrift/actions/runs/456"
		}
	}`)

	rec := doSignedRequest(t, h, payload, "secret", "workflow_run")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}

	select {
	case got := <-triggerCh:
		if got.Repository != "lambadalambda/eosrift" {
			t.Fatalf("repo = %q, want %q", got.Repository, "lambadalambda/eosrift")
		}
		if got.Workflow != "Docker Image" {
			t.Fatalf("workflow = %q, want %q", got.Workflow, "Docker Image")
		}
		if got.Branch != "main" {
			t.Fatalf("branch = %q, want %q", got.Branch, "main")
		}
		if got.SHA != "deadcafe" {
			t.Fatalf("sha = %q, want %q", got.SHA, "deadcafe")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("trigger was not called")
	}
}

func TestHandler_AllowsOnlySingleInFlightDeploy(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	unblock := make(chan struct{})
	h := mustHandler(t, Config{
		Secret:       "secret",
		WorkflowName: "Docker Image",
		Branch:       "main",
	}, func(context.Context, Run) error {
		calls.Add(1)
		<-unblock
		return nil
	})

	payload := []byte(`{
		"action":"completed",
		"repository":{"full_name":"lambadalambda/eosrift"},
		"workflow_run":{
			"id":999,
			"name":"Docker Image",
			"conclusion":"success",
			"head_branch":"main",
			"head_sha":"deadcafe",
			"html_url":"https://github.com/lambadalambda/eosrift/actions/runs/999"
		}
	}`)

	rec1 := doSignedRequest(t, h, payload, "secret", "workflow_run")
	if rec1.Code != http.StatusAccepted {
		t.Fatalf("first status = %d, want %d", rec1.Code, http.StatusAccepted)
	}

	// Give goroutine a moment to mark deploy as running.
	time.Sleep(50 * time.Millisecond)

	rec2 := doSignedRequest(t, h, payload, "secret", "workflow_run")
	if rec2.Code != http.StatusAccepted {
		t.Fatalf("second status = %d, want %d", rec2.Code, http.StatusAccepted)
	}

	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
	close(unblock)
}

func mustHandler(t *testing.T, cfg Config, trigger Trigger) *Handler {
	t.Helper()

	h, err := NewHandler(cfg, trigger, log.New(ioDiscard{}, "", 0))
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	return h
}

func doSignedRequest(t *testing.T, h http.Handler, payload []byte, secret, event string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/hooks/deploy", strings.NewReader(string(payload)))
	req.Header.Set("X-GitHub-Event", event)
	req.Header.Set("X-Hub-Signature-256", signPayload(t, payload, secret))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func signPayload(t *testing.T, payload []byte, secret string) string {
	t.Helper()

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
