package deployhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const maxPayloadBytes = 1 << 20 // 1 MiB

type Config struct {
	Secret       string
	WorkflowName string
	Branch       string
	Repository   string
	Timeout      time.Duration
}

type Run struct {
	Repository string
	Workflow   string
	Branch     string
	Conclusion string
	SHA        string
	URL        string
	RunID      int64
}

type Trigger func(ctx context.Context, run Run) error

type Handler struct {
	secret       []byte
	workflowName string
	branch       string
	repository   string
	timeout      time.Duration
	trigger      Trigger
	logger       *log.Logger
	running      atomic.Bool
}

type workflowRunEvent struct {
	Action     string `json:"action"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	WorkflowRun struct {
		ID         int64  `json:"id"`
		Name       string `json:"name"`
		Conclusion string `json:"conclusion"`
		HeadBranch string `json:"head_branch"`
		HeadSHA    string `json:"head_sha"`
		HTMLURL    string `json:"html_url"`
	} `json:"workflow_run"`
}

func NewHandler(cfg Config, trigger Trigger, logger *log.Logger) (*Handler, error) {
	if strings.TrimSpace(cfg.Secret) == "" {
		return nil, errors.New("secret is required")
	}
	if trigger == nil {
		return nil, errors.New("trigger is required")
	}
	if strings.TrimSpace(cfg.WorkflowName) == "" {
		cfg.WorkflowName = "Docker Image"
	}
	if strings.TrimSpace(cfg.Branch) == "" {
		cfg.Branch = "main"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Minute
	}
	if logger == nil {
		logger = log.Default()
	}

	return &Handler{
		secret:       []byte(cfg.Secret),
		workflowName: cfg.WorkflowName,
		branch:       cfg.Branch,
		repository:   strings.TrimSpace(cfg.Repository),
		timeout:      cfg.Timeout,
		trigger:      trigger,
		logger:       logger,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload, err := io.ReadAll(io.LimitReader(r.Body, maxPayloadBytes))
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	sig := strings.TrimSpace(r.Header.Get("X-Hub-Signature-256"))
	if !verifySignature(h.secret, payload, sig) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	event := strings.TrimSpace(r.Header.Get("X-GitHub-Event"))
	if event == "ping" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if event != "workflow_run" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var v workflowRunEvent
	if err := json.Unmarshal(payload, &v); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	run, ok, reason := h.match(v)
	if !ok {
		h.logger.Printf("deploy webhook ignored: %s", reason)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("ignored\n"))
		return
	}

	if !h.running.CompareAndSwap(false, true) {
		h.logger.Printf("deploy webhook ignored: already running run_id=%d", run.RunID)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("deploy already running\n"))
		return
	}

	go h.runTrigger(run)

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("accepted\n"))
}

func (h *Handler) runTrigger(run Run) {
	defer h.running.Store(false)

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	h.logger.Printf("deploy start repo=%s workflow=%s branch=%s sha=%s run_id=%d", run.Repository, run.Workflow, run.Branch, run.SHA, run.RunID)
	if err := h.trigger(ctx, run); err != nil {
		h.logger.Printf("deploy failed repo=%s run_id=%d err=%v", run.Repository, run.RunID, err)
		return
	}
	h.logger.Printf("deploy complete repo=%s run_id=%d", run.Repository, run.RunID)
}

func (h *Handler) match(v workflowRunEvent) (Run, bool, string) {
	if v.Action != "completed" {
		return Run{}, false, "action is not completed"
	}
	if v.WorkflowRun.Name != h.workflowName {
		return Run{}, false, "workflow name mismatch"
	}
	if v.WorkflowRun.Conclusion != "success" {
		return Run{}, false, "workflow did not succeed"
	}
	if v.WorkflowRun.HeadBranch != h.branch {
		return Run{}, false, "branch mismatch"
	}

	repo := strings.TrimSpace(v.Repository.FullName)
	if h.repository != "" && repo != h.repository {
		return Run{}, false, "repository mismatch"
	}

	return Run{
		Repository: repo,
		Workflow:   v.WorkflowRun.Name,
		Branch:     v.WorkflowRun.HeadBranch,
		Conclusion: v.WorkflowRun.Conclusion,
		SHA:        v.WorkflowRun.HeadSHA,
		URL:        v.WorkflowRun.HTMLURL,
		RunID:      v.WorkflowRun.ID,
	}, true, ""
}

func verifySignature(secret, payload []byte, got string) bool {
	if len(secret) == 0 {
		return false
	}
	const prefix = "sha256="
	if !strings.HasPrefix(got, prefix) {
		return false
	}

	sigHex := strings.TrimPrefix(got, prefix)
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	expected := mac.Sum(nil)
	return hmac.Equal(sig, expected)
}
