package config

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// HeaderAddList is a list of header additions in "Name: value" (or "Name=value")
// form.
//
// The YAML representation may be:
// - a sequence of strings
// - a mapping of string keys to string values
// - a sequence of single-entry mappings (common YAML gotcha for unquoted ":" strings)
type HeaderAddList []string

func (l *HeaderAddList) UnmarshalYAML(value *yaml.Node) error {
	if l == nil {
		return errors.New("HeaderAddList: nil receiver")
	}

	value = resolveYAMLAlias(value)
	if isYAMLNullNode(value) {
		*l = nil
		return nil
	}

	switch value.Kind {
	case yaml.ScalarNode:
		s := strings.TrimSpace(value.Value)
		if s == "" {
			*l = nil
			return nil
		}
		*l = HeaderAddList{s}
		return nil
	case yaml.SequenceNode:
		out := make([]string, 0, len(value.Content))
		for _, item := range value.Content {
			item = resolveYAMLAlias(item)
			if isYAMLNullNode(item) {
				continue
			}

			switch item.Kind {
			case yaml.ScalarNode:
				s := strings.TrimSpace(item.Value)
				if s == "" {
					return errors.New("invalid header entry: empty string")
				}
				out = append(out, s)
			case yaml.MappingNode:
				entries, err := headerStringsFromMappingNode(item)
				if err != nil {
					return err
				}
				out = append(out, entries...)
			default:
				return fmt.Errorf("invalid header entry: unsupported YAML node kind %d", item.Kind)
			}
		}
		*l = HeaderAddList(out)
		return nil
	case yaml.MappingNode:
		entries, err := headerStringsFromMappingNode(value)
		if err != nil {
			return err
		}
		*l = HeaderAddList(entries)
		return nil
	default:
		return fmt.Errorf("invalid header list: unsupported YAML node kind %d", value.Kind)
	}
}

func resolveYAMLAlias(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.AliasNode && n.Alias != nil {
		return n.Alias
	}
	return n
}

func isYAMLNullNode(n *yaml.Node) bool {
	if n == nil {
		return true
	}
	return n.Kind == yaml.ScalarNode && n.Tag == "!!null"
}

func headerStringsFromMappingNode(n *yaml.Node) ([]string, error) {
	if n == nil {
		return nil, errors.New("invalid header mapping: nil node")
	}
	if n.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("invalid header mapping: unsupported YAML node kind %d", n.Kind)
	}
	if len(n.Content)%2 != 0 {
		return nil, errors.New("invalid header mapping: odd number of mapping nodes")
	}

	out := make([]string, 0, len(n.Content)/2)
	for i := 0; i < len(n.Content); i += 2 {
		k := resolveYAMLAlias(n.Content[i])
		v := resolveYAMLAlias(n.Content[i+1])
		if k == nil || v == nil {
			return nil, errors.New("invalid header mapping: nil key or value node")
		}
		if k.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("invalid header mapping: unsupported key node kind %d", k.Kind)
		}
		if v.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("invalid header mapping: unsupported value node kind %d", v.Kind)
		}

		key := strings.TrimSpace(k.Value)
		val := strings.TrimSpace(v.Value)
		if key == "" {
			return nil, errors.New("invalid header mapping: empty key")
		}

		out = append(out, fmt.Sprintf("%s: %s", key, val))
	}

	return out, nil
}
