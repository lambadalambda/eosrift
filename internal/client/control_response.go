package client

import (
	"encoding/json"
	"io"
)

// readJSONControlResponse reads a control response from a yamux stream.
//
// Some peers may close a stream immediately after sending a valid JSON
// response, which can surface a non-EOF read error after payload delivery.
// If a complete JSON response is present, prefer that parsed response.
func readJSONControlResponse[T any](stream io.ReadCloser) (T, error) {
	var out T

	payload, readErr := io.ReadAll(stream)
	_ = stream.Close()

	if len(payload) == 0 {
		if readErr != nil {
			return out, readErr
		}
		return out, io.EOF
	}

	if err := json.Unmarshal(payload, &out); err != nil {
		if readErr != nil {
			return out, readErr
		}
		return out, err
	}

	// Parsed successfully; ignore transport errors that happened after payload.
	return out, nil
}
