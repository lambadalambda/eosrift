package inspect

import (
	"encoding/json"
	"net/http"
)

type listRequestsResponse struct {
	Requests []Entry `json:"requests"`
}

func Handler(store *Store) http.Handler {
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

	return mux
}

