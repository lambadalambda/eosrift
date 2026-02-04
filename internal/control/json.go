package control

import (
	"encoding/json"
	"io"
)

// WriteJSON writes v as a single JSON line (JSON + "\n").
//
// Prefer this over json.NewEncoder(w).Encode(v) for control-stream I/O; the
// encoder may perform multiple writes, which can surface spurious write errors
// when the peer closes quickly after decoding a request.
func WriteJSON(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = append(b, '\n')

	for len(b) > 0 {
		n, err := w.Write(b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}
