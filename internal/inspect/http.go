package inspect

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type listRequestsResponse struct {
	Requests []Entry `json:"requests"`
}

type HandlerOptions struct {
	Replay func(ctx context.Context, entry Entry) (ReplayResult, error)
}

type ReplayResult struct {
	StatusCode int `json:"status_code,omitempty"`
}

type replayResponse struct {
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

func Handler(store *Store, opts HandlerOptions) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("EosRift inspector (alpha)\n"))
	})

	mux.HandleFunc("/api/requests", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(listRequestsResponse{
			Requests: store.List(),
		})
	})

	mux.HandleFunc("/api/requests/", func(w http.ResponseWriter, r *http.Request) {
		// POST /api/requests/<id>/replay
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		rest := strings.TrimPrefix(r.URL.Path, "/api/requests/")
		parts := strings.Split(rest, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] != "replay" {
			http.NotFound(w, r)
			return
		}

		entry, ok := store.Get(parts[0])
		if !ok {
			http.NotFound(w, r)
			return
		}

		if opts.Replay == nil {
			http.Error(w, "replay not configured", http.StatusNotImplemented)
			return
		}

		res, err := opts.Replay(r.Context(), entry)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(replayResponse{
				Error: err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(replayResponse{
			StatusCode: res.StatusCode,
		})
	})

	return mux
}
