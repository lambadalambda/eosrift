package cli

import (
	"fmt"
	"strings"

	"eosrift.com/eosrift/internal/client"
	"eosrift.com/eosrift/internal/control"
)

func parseHeaderAddList(field string, values []string) ([]client.HeaderKV, error) {
	if len(values) == 0 {
		return nil, nil
	}

	out := make([]client.HeaderKV, 0, len(values))
	for _, raw := range values {
		kv, err := parseHeaderKV(field, raw)
		if err != nil {
			return nil, err
		}
		out = append(out, kv)
	}
	return out, nil
}

func parseHeaderRemoveList(field string, values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	out := make([]string, 0, len(values))
	for _, raw := range values {
		name, err := control.NormalizeHeaderName(field, raw)
		if err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, nil
}

func parseHeaderKV(field string, raw string) (client.HeaderKV, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return client.HeaderKV{}, fmt.Errorf("invalid %s: %q", field, raw)
	}

	name, value, ok := strings.Cut(s, ":")
	if !ok {
		name, value, ok = strings.Cut(s, "=")
	}
	if !ok {
		return client.HeaderKV{}, fmt.Errorf("invalid %s: %q", field, raw)
	}

	normName, err := normalizeHeaderName(field, name)
	if err != nil {
		return client.HeaderKV{}, err
	}

	normValue, err := control.ValidateHeaderValue(field, raw, value)
	if err != nil {
		return client.HeaderKV{}, err
	}

	return client.HeaderKV{
		Name:  normName,
		Value: normValue,
	}, nil
}

func normalizeHeaderName(field string, raw string) (string, error) {
	return control.NormalizeHeaderName(field, raw)
}
